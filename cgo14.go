// +build !go1.5

package gb

import (
	"go/build"
	"path/filepath"
)

// cgo support functions

// cgo returns a slice of post processed source files and an
// ObjTargets representing the result of compilation of the post .c
// output.
func cgo(pkg *Package) ([]ObjTarget, []string) {
	fn := func(t ...ObjTarget) ([]ObjTarget, []string) {
		return t, nil
	}
	if err := runcgo1(pkg); err != nil {
		return fn(ErrTarget{err})
	}

	defun, err := runcc(pkg, filepath.Join(pkg.Objdir(), "_cgo_defun.c"))
	if err != nil {
		return fn(ErrTarget{err})
	}

	cgofiles := []string{filepath.Join(pkg.Objdir(), "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(pkg.Objdir(), "_cgo_main.c"),
		filepath.Join(pkg.Objdir(), "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo2.c"))
	}

	var ofiles []string
	for _, f := range cfiles {
		ofile := stripext(f) + ".o"
		ofiles = append(ofiles, ofile)
		if err := rungcc1(pkg.Dir, ofile, f); err != nil {
			return fn(ErrTarget{err})
		}
	}

	ofile, err := rungcc2(pkg.Dir, ofiles)
	if err != nil {
		return fn(ErrTarget{err})
	}

	dynout, err := runcgo2(pkg, ofile)
	if err != nil {
		return fn(ErrTarget{err})
	}
	imports, err := runcc(pkg, dynout)
	if err != nil {
		return fn(ErrTarget{err})
	}

	allo, err := rungcc3(pkg.Dir, ofiles[1:]) // skip _cgo_main.o
	if err != nil {
		return fn(ErrTarget{err})
	}

	return []ObjTarget{cgoTarget(defun), cgoTarget(imports), cgoTarget(allo)}, cgofiles
}

type cgoTarget string

func (t cgoTarget) Objfile() string { return string(t) }
func (t cgoTarget) Result() error   { return nil }

// runcgo1 invokes the cgo tool to process pkg.CgoFiles.
func runcgo1(pkg *Package) error {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	if err := mkdir(objdir); err != nil {
		return err
	}

	args := []string{
		"-objdir", objdir,
		"--",
		"-I", pkg.Dir,
	}
	args = append(args, pkg.CgoFiles...)
	return run(pkg.Dir, cgo, args...)
}

// runcgo2 invokes the cgo tool to create _cgo_import.go
func runcgo2(pkg *Package, ofile string) (string, error) {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	dynout := filepath.Join(objdir, "_cgo_import.c")

	args := []string{
		"-objdir", objdir,
		"-dynimport", ofile,
		"-dynout", dynout,
	}
	return dynout, run(pkg.Dir, cgo, args...)
}

func runcc(pkg *Package, cfile string) (string, error) {
	archchar, err := build.ArchChar(pkg.GOARCH)
	if err != nil {
		return "", err
	}
	cc := filepath.Join(pkg.GOROOT, "pkg", "tool", pkg.GOOS+"_"+pkg.GOARCH, archchar+"c")
	objdir := pkg.Objdir()
	ofile := filepath.Join(stripext(cfile) + "." + archchar)
	args := []string{
		"-F", "-V", "-w",
		"-trimpath", pkg.Workdir(),
		"-I", objdir,
		"-I", filepath.Join(pkg.GOROOT, "pkg", pkg.GOOS+"_"+pkg.GOARCH), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_" + pkg.GOOS,
		"-D", "GOARCH_" + pkg.GOARCH,
		cfile,
	}
	return ofile, run(pkg.Dir, cc, args...)
}