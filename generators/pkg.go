package generators

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/runtime"
	gargs "k8s.io/gengo/args"
	"k8s.io/gengo/generator"
)

func Package(arguments *gargs.GeneratorArgs, name string, generators func(context *generator.Context) []generator.
Generator) generator.Package {
	boilerplate, err := arguments.LoadGoBoilerplate()
	runtime.Must(err)

	parts := strings.Split(name, "/")
	return &generator.DefaultPackage{
		PackageName:   groupPath(parts[len(parts)-1]),
		PackagePath:   name,
		HeaderText:    boilerplate,
		GeneratorFunc: generators,
	}
}

func groupPath(group string) string {
	g := strings.Replace(strings.Split(group, ".")[0], "-", "", -1)
	return groupPackageName(g, "")
}

func groupPackageName(group, groupPackageName string) string {
	if groupPackageName != "" {
		return groupPackageName
	}
	if group == "" {
		return "core"
	}
	return group
}
