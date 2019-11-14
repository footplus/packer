// Code generated by "mapstructure-to-hcl2 -type Config"; DO NOT EDIT.
package ansiblelocal

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

// FlatConfig is an auto-generated flat version of Config.
// Where the contents of a field with a `mapstructure:,squash` tag are bubbled up.
type FlatConfig struct {
	PackerBuildName     *string           `mapstructure:"packer_build_name" cty:"packer_build_name"`
	PackerBuilderType   *string           `mapstructure:"packer_builder_type" cty:"packer_builder_type"`
	PackerDebug         *bool             `mapstructure:"packer_debug" cty:"packer_debug"`
	PackerForce         *bool             `mapstructure:"packer_force" cty:"packer_force"`
	PackerOnError       *string           `mapstructure:"packer_on_error" cty:"packer_on_error"`
	PackerUserVars      map[string]string `mapstructure:"packer_user_variables" cty:"packer_user_variables"`
	PackerSensitiveVars []string          `mapstructure:"packer_sensitive_variables" cty:"packer_sensitive_variables"`
	Command             *string           `cty:"command"`
	ExtraArguments      []string          `mapstructure:"extra_arguments" cty:"extra_arguments"`
	GroupVars           *string           `mapstructure:"group_vars" cty:"group_vars"`
	HostVars            *string           `mapstructure:"host_vars" cty:"host_vars"`
	PlaybookDir         *string           `mapstructure:"playbook_dir" cty:"playbook_dir"`
	PlaybookFile        *string           `mapstructure:"playbook_file" cty:"playbook_file"`
	PlaybookFiles       []string          `mapstructure:"playbook_files" cty:"playbook_files"`
	PlaybookPaths       []string          `mapstructure:"playbook_paths" cty:"playbook_paths"`
	RolePaths           []string          `mapstructure:"role_paths" cty:"role_paths"`
	StagingDir          *string           `mapstructure:"staging_directory" cty:"staging_directory"`
	CleanStagingDir     *bool             `mapstructure:"clean_staging_directory" cty:"clean_staging_directory"`
	InventoryFile       *string           `mapstructure:"inventory_file" cty:"inventory_file"`
	InventoryGroups     []string          `mapstructure:"inventory_groups" cty:"inventory_groups"`
	GalaxyFile          *string           `mapstructure:"galaxy_file" cty:"galaxy_file"`
	GalaxyCommand       *string           `cty:"galaxy_command"`
}

// FlatMapstructure returns a new FlatConfig.
// FlatConfig is an auto-generated flat version of Config.
// Where the contents a fields with a `mapstructure:,squash` tag are bubbled up.
func (*Config) FlatMapstructure() interface{ HCL2Spec() map[string]hcldec.Spec } {
	return new(FlatConfig)
}

// HCL2Spec returns the hcl spec of a Config.
// This spec is used by HCL to read the fields of Config.
// The decoded values from this spec will then be applied to a FlatConfig.
func (*FlatConfig) HCL2Spec() map[string]hcldec.Spec {
	s := map[string]hcldec.Spec{
		"packer_build_name":          &hcldec.AttrSpec{Name: "packer_build_name", Type: cty.String, Required: false},
		"packer_builder_type":        &hcldec.AttrSpec{Name: "packer_builder_type", Type: cty.String, Required: false},
		"packer_debug":               &hcldec.AttrSpec{Name: "packer_debug", Type: cty.Bool, Required: false},
		"packer_force":               &hcldec.AttrSpec{Name: "packer_force", Type: cty.Bool, Required: false},
		"packer_on_error":            &hcldec.AttrSpec{Name: "packer_on_error", Type: cty.String, Required: false},
		"packer_user_variables":      &hcldec.BlockAttrsSpec{TypeName: "packer_user_variables", ElementType: cty.String, Required: false},
		"packer_sensitive_variables": &hcldec.AttrSpec{Name: "packer_sensitive_variables", Type: cty.List(cty.String), Required: false},
		"command":                    &hcldec.AttrSpec{Name: "command", Type: cty.String, Required: false},
		"extra_arguments":            &hcldec.AttrSpec{Name: "extra_arguments", Type: cty.List(cty.String), Required: false},
		"group_vars":                 &hcldec.AttrSpec{Name: "group_vars", Type: cty.String, Required: false},
		"host_vars":                  &hcldec.AttrSpec{Name: "host_vars", Type: cty.String, Required: false},
		"playbook_dir":               &hcldec.AttrSpec{Name: "playbook_dir", Type: cty.String, Required: false},
		"playbook_file":              &hcldec.AttrSpec{Name: "playbook_file", Type: cty.String, Required: false},
		"playbook_files":             &hcldec.AttrSpec{Name: "playbook_files", Type: cty.List(cty.String), Required: false},
		"playbook_paths":             &hcldec.AttrSpec{Name: "playbook_paths", Type: cty.List(cty.String), Required: false},
		"role_paths":                 &hcldec.AttrSpec{Name: "role_paths", Type: cty.List(cty.String), Required: false},
		"staging_directory":          &hcldec.AttrSpec{Name: "staging_directory", Type: cty.String, Required: false},
		"clean_staging_directory":    &hcldec.AttrSpec{Name: "clean_staging_directory", Type: cty.Bool, Required: false},
		"inventory_file":             &hcldec.AttrSpec{Name: "inventory_file", Type: cty.String, Required: false},
		"inventory_groups":           &hcldec.AttrSpec{Name: "inventory_groups", Type: cty.List(cty.String), Required: false},
		"galaxy_file":                &hcldec.AttrSpec{Name: "galaxy_file", Type: cty.String, Required: false},
		"galaxy_command":             &hcldec.AttrSpec{Name: "galaxy_command", Type: cty.String, Required: false},
	}
	return s
}
