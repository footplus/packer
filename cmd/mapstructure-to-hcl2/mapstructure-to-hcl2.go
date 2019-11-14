// mapstructure-to-hcl2 fills the gaps between hcl2 and mapstructure for Packer
//
// By generating a struct that the HCL2 ecosystem understands making use of
// mapstructure tags.
//
// Packer heavily uses the mapstructure decoding library to load/parse user
// config files. Packer now needs to move to HCL2.
//
// Here are a few differences/gaps betweens hcl2 and mapstructure:
//
//  * in HCL2 all basic struct fields (string/int/struct) that are not pointers
//   are required ( must be set ). In mapstructure everything is optional.
//
//  * mapstructure allows to 'squash' fields
//  (ex: Field CommonStructType `mapstructure:",squash"`) this allows to
//  decorate structs and reuse configuration code. HCL2 parsing libs don't have
//  anything similar.
//
// mapstructure-to-hcl2 will parse Packer's config files and generate the HCL2
// compliant code that will allow to not change any of the current builders in
// order to softly move to HCL2.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/structtag"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"golang.org/x/tools/go/packages"
)

var (
	typeNames  = flag.String("type", "", "comma-separated list of type names; must be set")
	output     = flag.String("output", "", "output file name; default srcdir/<type>_hcl2.go")
	trimprefix = flag.String("trimprefix", "", "trim the `prefix` from the generated constant names")
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of mapstructure-to-hcl2:\n")
	fmt.Fprintf(os.Stderr, "\tflatten-mapstructure [flags] -type T[,T...] pkg\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("mapstructure-to-hcl2: ")
	flag.Usage = Usage
	flag.Parse()
	if len(*typeNames) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	typeNames := strings.Split(*typeNames, ",")

	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}
	outputPath := strings.ToLower(typeNames[0]) + ".hcl2spec.go"
	if goFile := os.Getenv("GOFILE"); goFile != "" {
		outputPath = goFile[:len(goFile)-2] + "hcl2spec.go"
	}
	log.SetPrefix(fmt.Sprintf("mapstructure-to-hcl2: %s.%v: ", os.Getenv("GOPACKAGE"), typeNames))

	cfg := &packages.Config{
		Mode: packages.LoadSyntax,
	}
	pkgs, err := packages.Load(cfg, args...)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}
	topPkg := pkgs[0]
	sort.Strings(typeNames)

	var structs []StructDef
	usedImports := map[NamePath]*types.Package{}

	for id, obj := range topPkg.TypesInfo.Defs {
		if obj == nil {
			continue
		}
		t := obj.Type()
		nt, isANamedType := t.(*types.Named)
		if !isANamedType {
			continue
		}
		if nt.Obj().Pkg() != topPkg.Types {
			// Sometimes a struct embeds another struct named the same. ex:
			// builder/osc/bsuvolume.BlockDevice. This makes sure the type is
			// defined in topPkg.
			continue
		}
		ut := nt.Underlying()
		utStruct, utOk := ut.(*types.Struct)
		if !utOk {
			continue
		}
		pos := sort.SearchStrings(typeNames, id.Name)
		if pos >= len(typeNames) || typeNames[pos] != id.Name {
			continue // not a struct we care about
		}
		// make sure each type is found once where somehow sometimes they can be found twice
		typeNames = append(typeNames[:pos], typeNames[pos+1:]...)
		flatenedStruct := getMapstructureSquashedStruct(obj.Pkg(), utStruct)
		flatenedStruct = addCtyTagToStruct(flatenedStruct)
		newStructName := "Flat" + id.Name
		structs = append(structs, StructDef{
			OriginalStructName: id.Name,
			FlatStructName:     newStructName,
			Struct:             flatenedStruct,
		})

		for k, v := range getUsedImports(flatenedStruct) {
			if _, found := usedImports[k]; !found {
				usedImports[k] = v
			}
		}
	}

	out := bytes.NewBuffer(nil)

	fmt.Fprintf(out, `// Code generated by "mapstructure-to-hcl2 %s"; DO NOT EDIT.`, strings.Join(os.Args[1:], " "))
	fmt.Fprintf(out, "\npackage %s\n", topPkg.Name)

	delete(usedImports, NamePath{topPkg.Name, topPkg.PkgPath})
	usedImports[NamePath{"hcldec", "github.com/hashicorp/hcl/v2/hcldec"}] = types.NewPackage("hcldec", "github.com/hashicorp/hcl/v2/hcldec")
	usedImports[NamePath{"cty", "github.com/zclconf/go-cty/cty"}] = types.NewPackage("cty", "github.com/zclconf/go-cty/cty")
	outputImports(out, usedImports)

	sort.Slice(structs, func(i int, j int) bool {
		return structs[i].OriginalStructName < structs[j].OriginalStructName
	})
	for _, flatenedStruct := range structs {
		fmt.Fprintf(out, "\n// %s is an auto-generated flat version of %s.", flatenedStruct.FlatStructName, flatenedStruct.OriginalStructName)
		fmt.Fprintf(out, "\n// Where the contents of a field with a `mapstructure:,squash` tag are bubbled up.")
		fmt.Fprintf(out, "\ntype %s struct {\n", flatenedStruct.FlatStructName)
		outputStructFields(out, flatenedStruct.Struct)
		fmt.Fprint(out, "}\n")

		fmt.Fprintf(out, "\n// FlatMapstructure returns a new %s.", flatenedStruct.FlatStructName)
		fmt.Fprintf(out, "\n// %s is an auto-generated flat version of %s.", flatenedStruct.FlatStructName, flatenedStruct.OriginalStructName)
		fmt.Fprintf(out, "\n// Where the contents a fields with a `mapstructure:,squash` tag are bubbled up.")
		fmt.Fprintf(out, "\nfunc (*%s) FlatMapstructure() interface{ HCL2Spec() map[string]hcldec.Spec } {", flatenedStruct.OriginalStructName)
		fmt.Fprintf(out, "return new(%s)", flatenedStruct.FlatStructName)
		fmt.Fprint(out, "}\n")

		fmt.Fprintf(out, "\n// HCL2Spec returns the hcl spec of a %s.", flatenedStruct.OriginalStructName)
		fmt.Fprintf(out, "\n// This spec is used by HCL to read the fields of %s.", flatenedStruct.OriginalStructName)
		fmt.Fprintf(out, "\n// The decoded values from this spec will then be applied to a %s.", flatenedStruct.FlatStructName)
		fmt.Fprintf(out, "\nfunc (*%s) HCL2Spec() map[string]hcldec.Spec {\n", flatenedStruct.FlatStructName)
		outputStructHCL2SpecBody(out, flatenedStruct.Struct)
		fmt.Fprint(out, "}\n")
	}

	for impt := range usedImports {
		if strings.ContainsAny(impt.Path, "/") {
			out = bytes.NewBuffer(bytes.ReplaceAll(out.Bytes(),
				[]byte(impt.Path+"."),
				[]byte(impt.Name+".")))
		}
	}

	// avoid needing to import current pkg; there's probably a better way.
	out = bytes.NewBuffer(bytes.ReplaceAll(out.Bytes(),
		[]byte(topPkg.PkgPath+"."),
		nil))

	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("os.Create: %v", err)
	}

	_, err = outputFile.Write(goFmt(out.Bytes()))
	if err != nil {
		log.Fatalf("failed to write file: %v", err)
	}
}

type StructDef struct {
	OriginalStructName string
	FlatStructName     string
	Struct             *types.Struct
}

func outputStructHCL2SpecBody(w io.Writer, s *types.Struct) {
	fmt.Fprintf(w, "s := map[string]hcldec.Spec{\n")

	for i := 0; i < s.NumFields(); i++ {
		field, tag := s.Field(i), s.Tag(i)
		st, _ := structtag.Parse(tag)
		ctyTag, _ := st.Get("cty")
		fmt.Fprintf(w, "	\"%s\": ", ctyTag.Name)
		outputHCL2SpecField(w, ctyTag.Name, field.Type(), st)
		fmt.Fprintln(w, `,`)
	}

	fmt.Fprintln(w, `}`)
	fmt.Fprintln(w, `return s`)
}

func outputHCL2SpecField(w io.Writer, accessor string, fieldType types.Type, tag *structtag.Tags) {
	if m2h, err := tag.Get(""); err == nil && m2h.HasOption("self-defined") {
		fmt.Fprintf(w, `(&%s{}).HCL2Spec()`, fieldType.String())
		return
	}
	switch f := fieldType.(type) {
	case *types.Pointer:
		outputHCL2SpecField(w, accessor, f.Elem(), tag)
	case *types.Basic:
		fmt.Fprintf(w, `%#v`, &hcldec.AttrSpec{
			Name:     accessor,
			Type:     basicKindToCtyType(f.Kind()),
			Required: false,
		})
	case *types.Map:
		fmt.Fprintf(w, `%#v`, &hcldec.BlockAttrsSpec{
			TypeName:    accessor,
			ElementType: cty.String, // for now everything can be simplified to a map[string]string
			Required:    false,
		})
	case *types.Slice:
		elem := f.Elem()
		if ptr, isPtr := elem.(*types.Pointer); isPtr {
			elem = ptr.Elem()
		}
		switch elem := elem.(type) {
		case *types.Basic:
			fmt.Fprintf(w, `%#v`, &hcldec.AttrSpec{
				Name:     accessor,
				Type:     cty.List(basicKindToCtyType(elem.Kind())),
				Required: false,
			})
		case *types.Named:
			b := bytes.NewBuffer(nil)
			outputHCL2SpecField(b, accessor, elem, tag)
			fmt.Fprintf(w, `&hcldec.BlockListSpec{TypeName: "%s", Nested: %s}`, accessor, b.String())
		case *types.Slice:
			b := bytes.NewBuffer(nil)
			outputHCL2SpecField(b, accessor, elem.Underlying(), tag)
			fmt.Fprintf(w, `&hcldec.BlockListSpec{TypeName: "%s", Nested: %s}`, accessor, b.String())
		default:
			outputHCL2SpecField(w, accessor, elem.Underlying(), tag)
		}
	case *types.Named:
		underlyingType := f.Underlying()
		switch underlyingType.(type) {
		case *types.Struct:
			fmt.Fprintf(w, `&hcldec.BlockSpec{TypeName: "%s",`+
				` Nested: hcldec.ObjectSpec((*%s)(nil).HCL2Spec())}`, accessor, f.String())
		default:
			outputHCL2SpecField(w, f.String(), underlyingType, tag)
		}
	case *types.Struct:
		fmt.Fprintf(w, `&hcldec.BlockObjectSpec{TypeName: "%s",`+
			` Nested: hcldec.ObjectSpec((*%s)(nil).HCL2Spec())}`, accessor, fieldType.String())
	default:
		fmt.Fprintf(w, `%#v`, &hcldec.AttrSpec{
			Name:     accessor,
			Type:     basicKindToCtyType(types.Bool),
			Required: false,
		})
		fmt.Fprintf(w, `/* TODO(azr): could not find type */`)
	}
}

func basicKindToCtyType(kind types.BasicKind) cty.Type {
	switch kind {
	case types.Bool:
		return cty.Bool
	case types.String:
		return cty.String
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Float32, types.Float64,
		types.Complex64, types.Complex128:
		return cty.Number
	case types.Invalid:
		return cty.String // TODO(azr): fix that beforehand ?
	default:
		log.Printf("Un handled basic kind: %d", kind)
		return cty.String
	}
}

func outputStructFields(w io.Writer, s *types.Struct) {
	for i := 0; i < s.NumFields(); i++ {
		field, tag := s.Field(i), s.Tag(i)
		fieldNameStr := field.String()
		fieldNameStr = strings.Replace(fieldNameStr, "field ", "", 1)
		fmt.Fprintf(w, "	%s `%s`\n", fieldNameStr, tag)
	}
}

type NamePath struct {
	Name, Path string
}

func outputImports(w io.Writer, imports map[NamePath]*types.Package) {
	if len(imports) == 0 {
		return
	}
	// naive implementation
	pkgs := []NamePath{}
	for k := range imports {
		pkgs = append(pkgs, k)
	}
	sort.Slice(pkgs, func(i int, j int) bool {
		return pkgs[i].Path < pkgs[j].Path
	})

	fmt.Fprint(w, "import (\n")
	for _, pkg := range pkgs {
		if pkg.Name == pkg.Path || strings.HasSuffix(pkg.Path, "/"+pkg.Name) {
			fmt.Fprintf(w, "	\"%s\"\n", pkg.Path)
		} else {
			fmt.Fprintf(w, "	%s \"%s\"\n", pkg.Name, pkg.Path)
		}
	}
	fmt.Fprint(w, ")\n")
}

func getUsedImports(s *types.Struct) map[NamePath]*types.Package {
	res := map[NamePath]*types.Package{}
	for i := 0; i < s.NumFields(); i++ {
		fieldType := s.Field(i).Type()
		if p, ok := fieldType.(*types.Pointer); ok {
			fieldType = p.Elem()
		}
		if p, ok := fieldType.(*types.Slice); ok {
			fieldType = p.Elem()
		}
		namedType, ok := fieldType.(*types.Named)
		if !ok {
			continue
		}
		pkg := namedType.Obj().Pkg()
		res[NamePath{pkg.Name(), pkg.Path()}] = pkg
	}
	return res
}

func addCtyTagToStruct(s *types.Struct) *types.Struct {
	vars, tags := structFields(s)
	for i := range tags {
		field, tag := vars[i], tags[i]
		ctyAccessor := ToSnakeCase(field.Name())
		st, err := structtag.Parse(tag)
		if err == nil {
			if ms, err := st.Get("mapstructure"); err == nil && ms.Name != "" {
				ctyAccessor = ms.Name
			}
		}
		st.Set(&structtag.Tag{Key: "cty", Name: ctyAccessor})
		// st.Set(&structtag.Tag{Key: "hcl", Name: ctyAccessor, Options: []string{"optional"}})
		tags[i] = st.String()
	}
	return types.NewStruct(uniqueTags("cty", vars, tags))
}

func uniqueTags(tagName string, fields []*types.Var, tags []string) ([]*types.Var, []string) {
	outVars := []*types.Var{}
	outTags := []string{}
	uniqueTags := map[string]bool{}
	for i := range fields {
		field, tag := fields[i], tags[i]
		structtag, _ := structtag.Parse(tag)
		h, err := structtag.Get(tagName)
		if err == nil {
			if uniqueTags[h.Name] {
				log.Printf("skipping field %s ( duplicate `%s` %s tag  )", field.Name(), h.Name, tagName)
				continue
			}
			uniqueTags[h.Name] = true
		}
		outVars = append(outVars, field)
		outTags = append(outTags, tag)
	}
	return outVars, outTags
}

// getMapstructureSquashedStruct will return the same struct but embedded
// fields with a `mapstructure:",squash"` tag will be un-nested.
func getMapstructureSquashedStruct(topPkg *types.Package, utStruct *types.Struct) *types.Struct {
	res := &types.Struct{}
	for i := 0; i < utStruct.NumFields(); i++ {
		field, tag := utStruct.Field(i), utStruct.Tag(i)
		if !field.Exported() {
			continue
		}
		if _, ok := field.Type().(*types.Signature); ok {
			continue // ignore funcs
		}
		structtag, _ := structtag.Parse(tag)
		if ms, err := structtag.Get("mapstructure"); err != nil {
			//no mapstructure tag
		} else if ms.HasOption("squash") {
			ot := field.Type()
			uot := ot.Underlying()
			utStruct, utOk := uot.(*types.Struct)
			if !utOk {
				continue
			}

			res = squashStructs(res, getMapstructureSquashedStruct(topPkg, utStruct))
			continue
		}
		if field.Pkg() != topPkg {
			field = types.NewField(field.Pos(), topPkg, field.Name(), field.Type(), field.Embedded())
		}
		if p, isPointer := field.Type().(*types.Pointer); isPointer {
			// in order to make the following switch simpler we 'unwrap' this
			// pointer all structs are going to be made pointers anyways.
			field = types.NewField(field.Pos(), field.Pkg(), field.Name(), p.Elem(), field.Embedded())
		}
		switch f := field.Type().(type) {
		case *types.Named:
			switch f.String() {
			case "time.Duration":
				field = types.NewField(field.Pos(), field.Pkg(), field.Name(), types.NewPointer(types.Typ[types.String]), field.Embedded())
			case "github.com/hashicorp/packer/helper/config.Trilean": // TODO(azr): unhack this situation
				field = types.NewField(field.Pos(), field.Pkg(), field.Name(), types.NewPointer(types.Typ[types.Bool]), field.Embedded())
			case "github.com/hashicorp/packer/provisioner/powershell.ExecutionPolicy": // TODO(azr): unhack this situation
				field = types.NewField(field.Pos(), field.Pkg(), field.Name(), types.NewPointer(types.Typ[types.String]), field.Embedded())
			}
			if str, isStruct := f.Underlying().(*types.Struct); isStruct {
				obj := flattenNamed(f, str)
				field = types.NewField(field.Pos(), field.Pkg(), field.Name(), obj, field.Embedded())
				field = makePointer(field)
			}
			if slice, isSlice := f.Underlying().(*types.Slice); isSlice {
				if f, fNamed := slice.Elem().(*types.Named); fNamed {
					if str, isStruct := f.Underlying().(*types.Struct); isStruct {
						// this is a slice of named structs; we want to change
						// the struct ref to a 'FlatStruct'.
						obj := flattenNamed(f, str)
						slice := types.NewSlice(obj)
						field = types.NewField(field.Pos(), field.Pkg(), field.Name(), slice, field.Embedded())
					}
				}
			}
		case *types.Slice:
			if f, fNamed := f.Elem().(*types.Named); fNamed {
				if str, isStruct := f.Underlying().(*types.Struct); isStruct {
					obj := flattenNamed(f, str)
					field = types.NewField(field.Pos(), field.Pkg(), field.Name(), types.NewSlice(obj), field.Embedded())
				}
			}
		case *types.Basic:
			// since everything is optional, everything must be a pointer
			// non optional fields should be non pointers.
			field = makePointer(field)
		}
		res = addFieldToStruct(res, field, tag)
	}
	return res
}

func flattenNamed(f *types.Named, underlying types.Type) *types.Named {
	obj := f.Obj()
	obj = types.NewTypeName(obj.Pos(), obj.Pkg(), "Flat"+obj.Name(), obj.Type())
	return types.NewNamed(obj, underlying, nil)
}

func makePointer(field *types.Var) *types.Var {
	return types.NewField(field.Pos(), field.Pkg(), field.Name(), types.NewPointer(field.Type()), field.Embedded())
}

func addFieldToStruct(s *types.Struct, field *types.Var, tag string) *types.Struct {
	sf, st := structFields(s)
	return types.NewStruct(uniqueFields(append(sf, field), append(st, tag)))
}

func squashStructs(a, b *types.Struct) *types.Struct {
	va, ta := structFields(a)
	vb, tb := structFields(b)
	return types.NewStruct(uniqueFields(append(va, vb...), append(ta, tb...)))
}

func uniqueFields(fields []*types.Var, tags []string) ([]*types.Var, []string) {
	outVars := []*types.Var{}
	outTags := []string{}
	fieldNames := map[string]bool{}
	for i := range fields {
		field, tag := fields[i], tags[i]
		if fieldNames[field.Name()] {
			log.Printf("skipping duplicate %s field", field.Name())
			continue
		}
		fieldNames[field.Name()] = true
		outVars = append(outVars, field)
		outTags = append(outTags, tag)
	}
	return outVars, outTags
}

func structFields(s *types.Struct) (vars []*types.Var, tags []string) {
	for i := 0; i < s.NumFields(); i++ {
		field, tag := s.Field(i), s.Tag(i)
		vars = append(vars, field)
		tags = append(tags, tag)
	}
	return vars, tags
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func goFmt(b []byte) []byte {
	fb, err := format.Source(b)
	if err != nil {
		log.Printf("formatting err: %v", err)
		return b
	}
	return fb
}
