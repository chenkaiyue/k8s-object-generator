package args

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/types"
)

type CustomArgs struct {
	Package        string
	TypesByGroup   map[schema.GroupVersion][]*types.Name
	Options        Options
	OutputBase     string
	DestOutputBase string
}

type Group struct {
	Types            []interface{}
	GenerateTypes    bool
	PackageName      string
	ClientSetPackage string
	ListersPackage   string
	InformersPackage string
}

type Options struct {
	OutputPackage string
	OutputBase    string
	Groups        map[string]Group
	Boilerplate   string
	GenMocks      bool
}
