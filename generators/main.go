package generators

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	csargs "k8s.io/code-generator/cmd/client-gen/args"
	clientgenerators "k8s.io/code-generator/cmd/client-gen/generators"
	cs "k8s.io/code-generator/cmd/client-gen/generators"
	ctypes "k8s.io/code-generator/cmd/client-gen/types"
	dpargs "k8s.io/code-generator/cmd/deepcopy-gen/args"
	infargs "k8s.io/code-generator/cmd/informer-gen/args"
	inf "k8s.io/code-generator/cmd/informer-gen/generators"
	lsargs "k8s.io/code-generator/cmd/lister-gen/args"
	ls "k8s.io/code-generator/cmd/lister-gen/generators"
	gargs "k8s.io/gengo/args"
	dp "k8s.io/gengo/examples/deepcopy-gen/generators"
	"k8s.io/gengo/types"

	"github.com/mYmNeo/k8s-object-generator/args"
)

func Run(opts args.Options) {
	customArgs := &args.CustomArgs{
		Options:      opts,
		TypesByGroup: make(map[schema.GroupVersion][]*types.Name),
		Package:      opts.OutputPackage,
	}

	k8sArgs := gargs.Default().WithoutDefaultFlagParsing()
	k8sArgs.CustomArgs = customArgs
	k8sArgs.GoHeaderFilePath = opts.Boilerplate
	k8sArgs.InputDirs = getInputDirs(customArgs)

	if k8sArgs.OutputBase == "./" {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "can't create temp dir, %v", err)
			return
		}

		k8sArgs.OutputBase = tmpDir
		defer func(dir string) {
			_ = os.RemoveAll(dir)
		}(tmpDir)
	}
	customArgs.OutputBase = k8sArgs.OutputBase

	clientGenerator := NewClientGenerator()

	if err := k8sArgs.Execute(
		clientgenerators.NameSystems(),
		clientgenerators.DefaultNameSystem(),
		clientGenerator.Packages,
	); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}

	groups := map[string]bool{}
	for groupName, group := range customArgs.Options.Groups {
		if group.GenerateTypes {
			groups[groupName] = true
		}
	}

	if len(groups) == 0 {
		if err := copyGoPathToModules(customArgs); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "go modules copy failed: %v", err)
			os.Exit(1)
		}

		if opts.GenMocks {
			if err := clientGenerator.GenerateMocks(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "mocks failed: %v", err)
				os.Exit(1)
			}
		}

		return
	}

	if err := copyGoPathToModules(customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "go modules copy failed: %v", err)
		os.Exit(1)
	}

	if err := generateDeepcopy(groups, customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "deepcopy failed: %v", err)
		os.Exit(1)
	}

	if err := generateClientset(groups, customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "clientset failed: %v", err)
		os.Exit(1)
	}

	if err := generateListers(groups, customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "listers failed: %v", err)
		os.Exit(1)
	}

	if err := generateInformers(groups, customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "informers failed: %v", err)
		os.Exit(1)
	}

	if err := copyGoPathToModules(customArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "go modules copy failed: %v", err)
	}

	if opts.GenMocks {
		if err := clientGenerator.GenerateMocks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "mocks failed: %v", err)
			return
		}
	}
}

func getInputDirs(customArgs *args.CustomArgs) (inputDirs []string) {
	for gv, gen := range customArgs.Options.Groups {
		if gen.GenerateTypes {
			gen.InformersPackage = filepath.Join(customArgs.Package, "informers/externalverions")
			gen.ClientSetPackage = filepath.Join(customArgs.Package, "clientset/versioned")
			gen.ListersPackage = filepath.Join(customArgs.Package, "listers")
			customArgs.Options.Groups[gv] = gen
		}
	}

	for gv, gen := range customArgs.Options.Groups {
		args.ObjectsToGroupVersion(gv, gen.Types, customArgs.TypesByGroup)
	}

	for _, names := range customArgs.TypesByGroup {
		inputDirs = append(inputDirs, names[0].Package)
	}

	return
}

//until k8s code-gen supports gopath
func copyGoPathToModules(customArgs *args.CustomArgs) error {
	pathsToCopy := map[string]bool{}
	for _, tpg := range customArgs.TypesByGroup {
		for _, names := range tpg {
			pkg := sourcePackagePath(customArgs, names.Package)
			pathsToCopy[pkg] = true
		}
	}

	pkg := sourcePackagePath(customArgs, customArgs.Package)
	pathsToCopy[pkg] = true

	for pkg := range pathsToCopy {
		if _, err := os.Stat(pkg); os.IsNotExist(err) {
			continue
		}

		return filepath.Walk(pkg, func(path string, info os.FileInfo, err error) error {
			newPath := strings.Replace(path, pkg, ".", 1)
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				if info.IsDir() {
					return os.Mkdir(newPath, info.Mode())
				}

				return copyFile(path, newPath)
			}

			return err
		})
	}

	return nil
}

func sourcePackagePath(customArgs *args.CustomArgs, pkgName string) string {
	pkgSplit := strings.Split(pkgName, string(os.PathSeparator))
	pkg := filepath.Join(customArgs.OutputBase, strings.Join(pkgSplit[:3], string(os.PathSeparator)))
	return pkg
}

func copyFile(src, dst string) error {
	var err error
	var srcFd *os.File
	var dstFd *os.File
	var srcInfo os.FileInfo

	if srcFd, err = os.Open(src); err != nil {
		return err
	}
	defer srcFd.Close()

	if dstFd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstFd.Close()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return err
	}
	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func generateDeepcopy(groups map[string]bool, customArgs *args.CustomArgs) error {
	deepCopyCustomArgs := &dpargs.CustomArgs{}

	args := gargs.Default().WithoutDefaultFlagParsing()
	args.CustomArgs = deepCopyCustomArgs
	args.OutputBase = customArgs.OutputBase
	args.OutputFileBaseName = "zz_generated_deepcopy"
	args.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		args.InputDirs = append(args.InputDirs, names[0].Package)
		deepCopyCustomArgs.BoundingDirs = append(deepCopyCustomArgs.BoundingDirs, names[0].Package)
	}

	return args.Execute(dp.NameSystems(),
		dp.DefaultNameSystem(),
		dp.Packages)
}

func generateClientset(groups map[string]bool, customArgs *args.CustomArgs) error {
	arguments, clientSetArgs := csargs.NewDefaults()
	clientSetArgs.ClientsetName = "versioned"
	arguments.OutputBase = customArgs.OutputBase
	arguments.OutputPackagePath = filepath.Join(customArgs.Package, "clientset")
	arguments.GoHeaderFilePath = customArgs.Options.Boilerplate

	var order []schema.GroupVersion

	for gv := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		order = append(order, gv)
	}
	sort.Slice(order, func(i, j int) bool {
		return order[i].Group < order[j].Group
	})

	for _, gv := range order {
		packageName := customArgs.Options.Groups[gv.Group].PackageName
		if packageName == "" {
			packageName = gv.Group
		}
		names := customArgs.TypesByGroup[gv]
		arguments.InputDirs = append(arguments.InputDirs, names[0].Package)
		clientSetArgs.Groups = append(clientSetArgs.Groups, ctypes.GroupVersions{
			PackageName: packageName,
			Group:       ctypes.Group(gv.Group),
			Versions: []ctypes.PackageVersion{
				{
					Version: ctypes.Version(gv.Version),
					Package: names[0].Package,
				},
			},
		})
	}

	return arguments.Execute(cs.NameSystems(),
		cs.DefaultNameSystem(),
		cs.Packages)
}

func generateListers(groups map[string]bool, customArgs *args.CustomArgs) error {
	arguments, _ := lsargs.NewDefaults()
	arguments.OutputBase = customArgs.OutputBase
	arguments.OutputPackagePath = filepath.Join(customArgs.Package, "listers")
	arguments.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		arguments.InputDirs = append(arguments.InputDirs, names[0].Package)
	}

	return arguments.Execute(ls.NameSystems(),
		ls.DefaultNameSystem(),
		ls.Packages)
}

func generateInformers(groups map[string]bool, customArgs *args.CustomArgs) error {
	arguments, clientSetArgs := infargs.NewDefaults()
	clientSetArgs.VersionedClientSetPackage = filepath.Join(customArgs.Package, "clientset/versioned")
	clientSetArgs.ListersPackage = filepath.Join(customArgs.Package, "listers")
	arguments.OutputBase = customArgs.OutputBase
	arguments.OutputPackagePath = filepath.Join(customArgs.Package, "informers")
	arguments.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		arguments.InputDirs = append(arguments.InputDirs, names[0].Package)
	}

	return arguments.Execute(inf.NameSystems(),
		inf.DefaultNameSystem(),
		inf.Packages)
}
