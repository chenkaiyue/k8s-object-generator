package generators

import (
	"fmt"

	gargs "k8s.io/gengo/args"
	"k8s.io/gengo/generator"

	"github.com/mYmNeo/k8s-object-generator/args"
)

func RegisterGroupGo(group string, args *gargs.GeneratorArgs, customArgs *args.CustomArgs) generator.Generator {
	return &registerGroupGo{
		group:      group,
		args:       args,
		customArgs: customArgs,
		DefaultGen: generator.DefaultGen{
			OptionalName: "zz_generated_register",
		},
	}
}

type registerGroupGo struct {
	generator.DefaultGen

	group      string
	args       *gargs.GeneratorArgs
	customArgs *args.CustomArgs
}

func (f *registerGroupGo) PackageConsts(*generator.Context) []string {
	return []string{
		fmt.Sprintf("GroupName = \"%s\"", f.group),
	}
}
