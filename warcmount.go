// Hellofs implements a simple "hello world" file system.
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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

	rt, err := newFS(rdr)
	if err != nil {
		log.Fatal(err)
	}

	mountdir := filepath.Base(os.Args[1])
	mountdir = strings.TrimSuffix(mountdir, filepath.Ext(mountdir))
	mountdir = strings.TrimSuffix(mountdir, filepath.Ext(mountdir))
	err = os.MkdirAll(mountdir, 0777)
	if err != nil {
		log.Fatal(err)
	}

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

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		<-sc
		fuse.Unmount(mountdir)
	}()

	fmt.Printf("Mounting %s at %s, use ctrl-c to unmount\n", os.Args[1], mountdir)
	fs.Serve(c, rt)
	os.Remove(mountdir)
	os.Exit(1)
}

type FS struct {
	files []*File
}

func newFS(rdr webarchive.Reader) (*FS, error) {
	repl := strings.NewReplacer("\\", "_", "/", "_")
	files := make([]*File, 0, 100)
	idx := 2
	var record webarchive.Record
	var err error
	for record, err = rdr.NextPayload(); err == nil; record, err = rdr.NextPayload() {
		byt, e := ioutil.ReadAll(record)
		if e != nil {
			return nil, e
		}
		files = append(files, &File{
			idx:     uint64(idx),
			sz:      uint64(record.Size()),
			name:    repl.Replace(record.URL()),
			content: byt,
		})
		idx++
	}
	if err == io.EOF {
		err = nil
	}
	return &FS{files}, err
}

func (f *FS) Root() (fs.Node, error) {
	return &Root{f}, nil
}

type Root struct {
	*FS
}

func (r *Root) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (r *Root) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, v := range r.files {
		if v.name == name {
			return v, nil
		}
	}
	return nil, fuse.ENOENT
}

func (r *Root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirs := make([]fuse.Dirent, len(r.files))
	for i, v := range r.files {
		dirs[i] = fuse.Dirent{
			Inode: v.idx,
			Name:  v.name,
			Type:  fuse.DT_File,
		}
	}
	return dirs, nil
}

type File struct {
	idx     uint64
	sz      uint64
	name    string
	content []byte
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.idx
	a.Mode = 0444
	a.Size = f.sz
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	return f.content, nil
}
