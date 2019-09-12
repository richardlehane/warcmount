// Hellofs implements a simple "hello world" file system.
package main

import (
	"io/ioutil"
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
	os.MkdirAll(mountdir, 0777)

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

	err = fs.Serve(c, FS{Dir{f, rdr}})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	nd fs.Node
}

func (f FS) Root() (fs.Node, error) {
	return f.nd, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	f *os.File
	r webarchive.Reader
}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.f.Seek(0, 0)
	d.r.Reset(d.f)
	var idx uint64 = 2
	for record, err := d.r.NextPayload(); err == nil; record, err = d.r.NextPayload() {
		if name == record.URL() {
			return File{idx, record}, nil
		}
		idx++
	}
	return nil, fuse.ENOENT
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.f.Seek(0, 0)
	d.r.Reset(d.f)
	var idx uint64 = 2
	dirs := make([]fuse.Dirent, 0, 100)
	for record, err := d.r.NextPayload(); err == nil; record, err = d.r.NextPayload() {
		dirs = append(dirs, fuse.Dirent{
			Inode: idx,
			Name:  record.URL(),
			Type:  fuse.DT_File,
		})
		idx++
	}
	return dirs, nil
}

type File struct {
	idx uint64
	rec webarchive.Record
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.idx
	a.Mode = 0444
	a.Size = uint64(f.rec.Size())
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	return ioutil.ReadAll(f.rec)
}
