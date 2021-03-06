package generators

import (
	"io"

	"k8s.io/apimachinery/pkg/runtime/schema"
	gargs "k8s.io/gengo/args"
	"k8s.io/gengo/generator"

	"github.com/mYmNeo/k8s-object-generator/args"
)

func ListTypesGo(gv schema.GroupVersion, args *gargs.GeneratorArgs, customArgs *args.CustomArgs) generator.Generator {
	return &listTypesGo{
		gv:         gv,
		args:       args,
		customArgs: customArgs,
		DefaultGen: generator.DefaultGen{
			OptionalName: "zz_generated_list_types",
		},
	}
}

type listTypesGo struct {
	generator.DefaultGen

	gv         schema.GroupVersion
	args       *gargs.GeneratorArgs
	customArgs *args.CustomArgs
}

func (f *listTypesGo) Imports(*generator.Context) []string {
	packages := []string{
		"metav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"",
	}

	return packages
}

func (f *listTypesGo) Init(c *generator.Context, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	for _, t := range f.customArgs.TypesByGroup[f.gv] {
		m := map[string]interface{}{
			"type": t.Name,
		}
		args.CheckType(c.Universe.Type(*t))
		sw.Do(string(listTypesBody), m)
	}

	return sw.Error()
}

var listTypesBody = `
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.type}}List is a list of {{.type}} resources
type {{.type}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata"` + "`" + `

	Items []{{.type}} ` + "`" + `json:"items"` + "`" + `
}

func New{{.type}}(namespace, name string, obj {{.type}}) *{{.type}} {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("{{.type}}").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}
`
