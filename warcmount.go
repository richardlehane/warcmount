// Hellofs implements a simple "hello world" file system.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"

	"github.com/richardlehane/webarchive"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Usage: warcmount WARC_FILE.warc.gz")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	rdr, err := webarchive.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}

	mountdir := filepath.Base(os.Args[1])
	mountdir = strings.TrimSuffix(mountdir, filepath.Ext(mountdir))
	mountdir = strings.TrimSuffix(mountdir, filepath.Ext(mountdir))

	c, err := fuse.Mount(
		mountdir,
		fuse.FSName("warcmount"),
		fuse.Subtype("warcmountfs"),
		fuse.LocalVolume(),
		fuse.VolumeName(mountdir),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, FS{})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// FS implements the hello world file system.
type FS struct{}

func (FS) Root() (fs.Node, error) {
	return Dir{}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct{}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == "hello" {
		return File{}, nil
	}
	return nil, fuse.ENOENT
}

var dirDirs = []fuse.Dirent{
	{Inode: 2, Name: "hello", Type: fuse.DT_File},
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return dirDirs, nil
}

// File implements both Node and Handle for the hello file.
type File struct{}

const greeting = "hello, world\n"

func (File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 2
	a.Mode = 0444
	a.Size = uint64(len(greeting))
	return nil
}

func (File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}
