// Code generated by go-bindata.
// sources:
// tmpl/changelog.html
// tmpl/index.html
// DO NOT EDIT!

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _tmplChangelogHtml = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xb4\x55\x4b\x6f\xe4\x36\x0c\x3e\x27\xbf\x82\xf5\xdd\xd2\xe4\xd1\xa6\x48\x1d\x03\x41\x7a\xc8\xa1\x48\x81\x76\xb1\x77\xd9\xa2\xc7\x9a\x68\x24\x47\x92\xe7\x81\x20\xff\x7d\x69\xf9\x91\x19\x63\xb2\xd9\xc3\xee\xc9\xa2\x48\x7e\x1f\xc9\x4f\xc3\xc9\xea\xb0\xd6\xf9\xf9\x59\x56\xa3\x90\xf4\x3d\xcb\xb4\x32\xcf\xe0\x50\xdf\x25\x3e\xec\x35\xfa\x1a\x31\x24\x50\x3b\xac\xee\x92\x3a\x84\xc6\xdf\x72\xbe\x16\xbb\x52\x1a\x56\x58\x1b\x7c\x70\xa2\xe9\x8c\xd2\xae\xf9\x74\xc1\xaf\xd8\x15\xbb\xe4\xa5\xf7\xef\x77\x6c\xad\x28\xca\xfb\xe4\xd7\xd2\xa4\xa1\xc6\x35\xfe\x3c\xb2\xca\x9a\x90\x8a\x2d\x7a\xbb\x46\x7e\xcd\x7e\x67\x8b\xc8\x77\x78\x7d\x40\xd6\xb1\x45\x8e\xfc\xbc\xb0\x72\x0f\xaf\xd0\x08\x29\x95\x59\xa6\xc1\x36\xb7\x70\xb3\x68\x76\x7f\xc1\x5b\x17\xc5\x87\xb0\x98\x51\x3a\xd5\x04\xf0\xae\x7c\xaf\x87\xf8\x57\x9e\x95\xda\xb6\xb2\xd2\xc2\x61\x2c\x46\xac\xc4\x8e\x6b\x55\x78\xbe\x7a\x69\xd1\xed\xf9\x25\xbb\x60\xd7\x83\x11\xcb\x58\x51\x15\x84\x1d\x01\xf3\x8f\xb0\x7f\x74\xb0\xab\xb9\x7c\x33\xf4\x8c\x0f\xcf\x26\xeb\x9a\x8d\x74\x52\x6d\xa0\xd4\xc2\xfb\xbb\xc4\x88\x4d\x21\x1c\xf4\x9f\x54\x99\x0d\x3a\x8f\xa3\x59\xa9\x1d\xca\x6e\x28\x51\xa1\xa3\xbc\x92\x46\x2b\x94\x41\xd7\xbb\x4e\x60\xa6\x1d\xed\xe4\x3f\xcb\xc4\xcc\x5d\x38\x61\xe4\x28\x30\x4f\xf2\x47\x14\x2e\x90\xd5\x68\xb1\x87\x07\xad\xd0\x84\x8c\x8b\x01\x9d\x13\xfc\x70\x6c\xf5\x01\xd0\x58\x29\x7d\x26\x22\xad\x72\x22\x1b\x70\xa9\xce\x4a\x2d\x93\xfc\x3f\xd4\x56\x48\x78\x88\x66\x87\x9b\x91\x44\xa7\x33\x6a\x61\x96\x14\x4d\x49\x0f\xe3\xf1\x38\x21\xe3\xad\xfe\xa4\x98\xf1\xe8\xd4\xb2\x0e\x47\x95\xf9\x46\x98\xd9\x24\x02\xee\x28\xe6\x2b\x4d\x5e\x59\x73\x0b\xaf\xaf\xc0\x06\x03\xde\xde\x48\x49\xca\x38\xc5\x3e\x0d\x65\x3a\x50\xa6\xaa\xc0\xe0\x7b\x3e\xfb\x47\x04\xf4\x81\x70\xce\xbf\xab\x5f\xd6\x8c\x8e\x62\x99\xca\xae\x6b\x07\x5d\x59\x69\x49\x32\x74\x41\x4f\xb8\x85\x11\xf4\x7e\x23\x94\x16\x85\xc6\xdf\x60\x1a\x5b\xf7\x6a\xe9\xd1\xb6\x8d\x24\x42\x56\x1f\x68\x19\x1f\x6d\xaf\x67\xda\xb5\x36\x55\x14\xad\x7f\xff\x1f\x4f\xf7\xae\xac\xe9\xdc\xf7\x80\x2f\xc3\x45\xb2\x55\x46\xda\xad\x4f\xc8\xc5\x70\x87\xe4\x46\x23\xc9\x48\xf2\xbf\xed\xd6\x44\x51\x1f\x91\x7e\x79\xbd\x44\xcd\x7c\x1e\x7d\xf0\x27\xcd\xcb\x51\x4e\x19\xf2\xf9\xf4\x65\x18\x7d\x72\xd2\xfc\x40\x4f\xf8\x42\xcd\xd0\xea\x00\xda\x68\xd0\x77\xef\x18\x9c\x7c\x2b\xbc\x43\x98\x78\xfe\xb8\x90\xe5\xe2\x46\x1c\x10\x1c\xe1\xde\x4b\x89\x12\x70\x5d\x60\xdc\x4c\x60\x2b\xda\x8d\xde\xb6\xae\x44\xcf\x66\x6f\xf7\xc9\x6e\x41\xb4\xc1\xa6\x91\xbf\x8b\xf6\xb5\x6d\xb5\x84\xad\x75\xcf\xd0\x38\xdb\xa0\xd3\x7b\x36\x2f\x6a\x2a\x45\x5c\x5c\x2d\xfe\xac\x16\x1f\xf4\x0a\xb1\xcd\xbe\x9e\xe9\xb7\x41\x1b\x73\x89\x43\x9f\x30\xef\xb1\x3f\xe8\x43\x29\x32\xde\x2f\x1f\x5a\x46\xf1\xbf\xec\x5b\x00\x00\x00\xff\xff\xd2\xe6\xea\xf5\xd3\x06\x00\x00")

func tmplChangelogHtmlBytes() ([]byte, error) {
	return bindataRead(
		_tmplChangelogHtml,
		"tmpl/changelog.html",
	)
}

func tmplChangelogHtml() (*asset, error) {
	bytes, err := tmplChangelogHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "tmpl/changelog.html", size: 1747, mode: os.FileMode(420), modTime: time.Unix(1454242289, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _tmplIndexHtml = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xb4\x57\x6d\x6f\xdb\x36\x10\xfe\x6c\xff\x0a\x42\x1b\x2a\x19\x8d\xa9\xb4\xe9\x36\xc0\x91\x05\x14\x2b\xb6\x62\xeb\xba\x62\xe9\x56\xec\xd3\x40\x8b\xb4\xc5\x84\x22\x35\x92\x8e\x9d\x05\xfe\xef\x3b\xbe\xc8\x96\xed\xa4\x0e\x8a\x2d\x08\x6c\xf2\x78\xbc\x97\x87\xcf\x1d\xe9\xa2\xb6\x8d\x28\x87\x83\xa2\x66\x84\xc2\xf7\xa0\x10\x5c\xde\x20\xcd\xc4\x34\x31\xf6\x4e\x30\x53\x33\x66\x13\x54\x6b\x36\x9f\x26\xb5\xb5\xad\x99\xe4\x79\x43\xd6\x15\x95\x78\xa6\x94\x35\x56\x93\xd6\x4d\x2a\xd5\xe4\x5b\x41\x7e\x81\x2f\xf0\xcb\xbc\x32\x66\x27\xc3\x0d\x07\x2d\x63\x92\xff\xd7\xcd\xd8\xd6\xac\x61\xff\x9d\xb3\xb9\x92\x76\x4c\x56\xcc\xa8\x86\xe5\xaf\xf0\x37\xf8\xdc\xfb\xeb\x8b\x7b\xce\x9c\x37\xef\xa3\x1c\xce\x14\xbd\x43\xf7\xa8\x25\x94\x72\xb9\x18\x5b\xd5\x4e\xd0\x77\xe7\xed\xfa\x12\x6d\x9c\x56\x1e\xd5\xfc\x8e\x4a\xf3\xd6\x22\xa3\xab\x5d\x3c\xe0\xff\xda\xe0\x4a\xa8\x25\x9d\x0b\xa2\x99\x0f\x86\x5c\x93\x75\x2e\xf8\xcc\xe4\xd7\x7f\x2f\x99\xbe\xcb\x5f\xe2\x17\xf8\x55\x9c\xf8\x30\xae\x21\x0a\xb0\xed\x0d\x96\x8f\xd9\x7e\x2a\xb0\xd7\x87\xc7\xf7\x24\xeb\x27\x22\x6f\x00\x31\x69\xc1\x94\x0b\xfe\x1c\x7f\xdb\x09\x8e\xed\xef\x1c\x94\xc3\x5b\xa2\x91\x61\xfa\x96\x69\xa3\xaa\x1b\x66\xd1\x14\x49\xb6\x42\x9f\xd8\xec\xca\xcf\xb3\x64\xe5\x7c\x0b\x55\x11\x51\x2b\x63\x27\xf7\xf7\x08\x7f\x50\xda\xa2\xcd\x06\xa4\x0b\x93\x8c\x2e\x87\x7d\x03\x58\xc9\x86\x19\x43\x16\x0c\x4c\xcd\x97\xb2\xb2\x5c\xc9\x8c\x8d\xd0\xfd\x70\xe0\x9c\x51\x10\xff\x74\xf5\xeb\x7b\xdc\x12\x6d\x58\xc6\x30\x25\x96\x80\x0d\xbf\xb8\x86\xc5\xaf\xb3\xf4\xab\x14\x3d\x47\x14\xff\xcc\xee\xdc\x02\x9f\xa3\x6c\x8d\x05\x93\x0b\x5b\x7b\x33\x5e\xd5\x15\x18\x68\x27\xc9\x19\x08\x06\x95\x1f\x53\x22\x17\x4c\x07\x09\xaf\x94\x74\x32\xc0\xcb\xb0\xa8\x04\xdc\x02\x44\x40\x4a\xf1\x6f\x8c\x18\x25\xbd\xd8\xb3\x78\x8a\xd2\x82\x1c\x50\x17\x6a\x57\x5b\x10\xb5\x82\xdc\x79\xb0\x17\x04\x52\xcb\x43\x70\xbf\xb7\x42\x11\xca\x34\x4c\xd2\x7c\x1b\xaf\x9f\x25\xe5\x2d\x67\xab\x22\x27\x65\x0a\xe1\x0f\xcc\x8a\xdb\xaa\x46\x19\xc5\x57\x96\xd8\xa5\x09\x39\x0c\x2a\x62\x18\x4a\xae\x96\x55\x05\x70\x25\x13\x27\x8a\x69\x98\x28\xbb\xf4\xb2\x6d\x22\x35\xab\x6e\xa2\x6c\x97\x89\x0b\x3e\xc8\x66\x9a\x91\x30\x8c\x96\x6f\x78\xdb\x32\xba\x67\x79\x45\xb4\x84\xa2\x79\x92\xe5\xf4\xb5\x00\x8b\x50\x6b\x31\x53\x3a\x79\x26\x67\xa6\xbd\x74\xb9\x3e\xe0\xd5\x15\x5f\x3c\x93\xb4\x30\x2d\x91\xa8\x12\xc4\x98\x69\x62\xd9\xda\x8e\xdd\x26\x08\xe1\x79\x0a\x3c\xe4\xdd\xca\x9c\xa0\x39\xf1\x4b\x3e\x12\x80\xce\xb1\x94\x97\x3b\x3f\x5d\x34\xb0\x04\xf4\x05\xa3\x01\xd1\x3d\x8a\xb8\xc5\xb1\xf1\xc8\xa6\x23\xec\x42\xc8\xdc\x87\x63\xce\x06\x31\x01\x50\x74\x94\xb1\x3a\xd0\xab\xb0\xba\x04\x55\x62\xad\xce\x52\x4e\xd3\xb3\x2d\xd7\x40\xd1\xb1\x8d\xe2\x0f\x70\xe4\xda\x60\x4f\x7b\xfc\x89\x4b\xc9\x74\x3c\x36\xab\xe3\x46\x9f\x04\xec\x4d\xe3\x71\xa5\x23\x0f\xc3\xce\xe5\x03\xaa\x81\xa0\x51\x73\x18\xc3\x32\x10\x55\xa8\xd5\x40\x11\x6d\xfd\xba\xdb\x0d\x07\x28\x69\xe6\x43\xa6\x65\x97\xdd\x41\x74\x6f\x99\x56\xa3\xcf\xef\x08\x07\x52\x06\xc8\xe2\x66\xcd\x1a\x65\x99\xdf\xdd\x83\x37\x40\x5f\x98\x86\x08\xb1\x77\x80\xcd\xd2\x02\x97\xca\xec\x21\x1b\xef\xa1\x2c\x9c\x8d\x51\x77\x46\x27\xc2\xa1\xf8\x0d\x37\xae\xa8\x3e\xde\xb5\xec\x84\xae\xc1\x73\xa5\x1b\x02\x7d\x88\x52\x7a\x86\x7e\x81\x3f\xf4\x46\x9d\xa1\x7a\xd2\x34\x88\x24\xa3\x47\xb6\x9f\xc8\x00\xac\x6a\xd5\xbc\x57\xab\x6c\xd4\x05\xee\xf4\x4f\x46\x1e\x5b\x2a\x5d\x6a\xe2\xdb\x1a\x64\x12\x87\x28\x47\x2f\xce\xfd\x1f\x1c\x73\xc3\x85\xe0\x86\x01\x7b\xa9\xe7\xe4\xb2\x21\x92\xff\xc3\xb2\x18\x6d\xa0\x58\x68\x06\x68\x3a\x85\x0a\x0c\x25\xe6\x2a\xb3\xc7\xb2\xc3\x08\x0e\xf9\xfa\x00\xf1\xd3\xc3\xe2\x32\xad\xe7\x6e\x37\x0c\xf5\x15\xb3\x3c\xa4\xea\x17\xf9\x3b\xaa\xf4\xae\xc5\x1c\xd5\x79\x68\xc4\x07\x05\xde\x35\xe2\x7e\x85\x8f\x76\xe5\xe1\x0a\xdd\x37\x5d\x70\xd8\x42\x23\x76\x01\x5a\xbd\x05\xd1\x2d\x2f\xe1\xf2\xf9\xcb\xbd\xb2\x5c\x5d\xe1\xaa\xe6\x82\x6a\x26\xb3\x51\xbc\x30\x3c\xbe\xe7\xe8\xd9\x33\xb4\x87\x78\xba\x45\x3c\x8d\x88\x1f\xdb\x8a\x19\x7e\xf9\xad\x90\x94\x7f\xc0\x3d\x80\xde\x71\x63\x91\x9a\xa3\x3f\xd5\x52\xa3\x1f\x9d\xb2\xbf\x1a\x62\x96\xf0\xbf\x39\xba\x41\x3d\x56\xc7\xf7\xa7\x8b\xb1\xd5\x6a\xa1\x7d\xbb\xc1\x0e\xef\x2c\xf9\x81\x4b\x0e\x2f\x2f\xea\x2e\xe2\xcd\x65\x78\x02\x75\x0f\x89\x22\x8f\xef\xcf\xc2\xbd\x9a\xfc\xcb\x82\xf2\xdb\xee\x5c\x24\xb9\x9d\x41\xff\x09\x5f\x63\x2e\x5d\x04\xac\x9b\xce\xf9\x9a\x51\xf7\xba\xf2\x4f\xbd\xbd\x7d\xae\x2b\x13\x0e\xb4\x0a\x4b\x0f\xd8\x1c\x07\x10\xe3\xfa\x00\x30\xdc\x5f\x9e\x69\x22\x69\xf7\x52\x04\x9c\xde\xf6\x30\x45\xdf\x0b\x0e\x45\xe6\x30\x0a\xd6\x73\x30\x1f\x87\x4b\xd1\x33\xd4\x45\xda\x1b\x6a\xbe\xa8\xed\xd6\xab\xe0\x07\x6e\x1d\x60\x70\x28\x90\x26\x60\x3a\x41\xee\x29\x13\x27\xf0\x9a\x29\xe0\x19\xd5\x79\x5c\x8a\x90\x74\xe7\x7a\x37\x78\x1c\x06\xbf\xc4\xe9\x34\xe9\xb1\x28\xd9\x2b\x8d\x0a\xd2\xda\x82\x02\xce\x81\xc2\xbe\x93\x02\x5d\x36\x9b\x61\x04\xea\x49\x64\xf3\x8f\xb0\x6e\xe7\xe7\x79\xd6\x39\x83\xd2\x89\x5e\x76\x80\x16\xf5\x45\xf9\xb1\xe6\x06\x5d\x01\xa1\x3c\x24\xc7\xf5\x1c\xda\xa6\x4f\xac\xa3\x5e\x52\x6e\xab\x07\x63\x1c\xcb\x16\xa8\x76\x11\x8c\x5a\x32\x13\x6c\x6b\xc3\x4f\xfc\xe7\xb8\x56\xb7\x3b\xce\xd8\xee\x97\x91\x9f\xe8\x38\x72\xf2\x1d\xbe\x62\xdc\xd0\xf1\x8b\xa4\x7c\x0d\x99\xd8\xfa\x71\x95\x97\xa0\xb2\x80\xa3\x30\xf6\xf3\x7a\x60\xca\x5d\x39\x47\x4a\x4a\xb8\x14\xa6\x09\xd8\xf1\xf7\xef\x49\x2b\xef\x7c\x6f\x39\x19\x54\x68\x38\x3d\x35\x18\xea\x8e\x63\x3d\x00\x0a\xeb\x7f\xd4\x38\x8c\xfd\xf9\x26\x5b\x9d\xae\x6c\xdd\xd8\x61\xd8\x27\x63\x91\x87\x55\x40\xde\xff\xd8\xfc\x37\x00\x00\xff\xff\xc0\x1b\x72\x13\x74\x0e\x00\x00")

func tmplIndexHtmlBytes() ([]byte, error) {
	return bindataRead(
		_tmplIndexHtml,
		"tmpl/index.html",
	)
}

func tmplIndexHtml() (*asset, error) {
	bytes, err := tmplIndexHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "tmpl/index.html", size: 3700, mode: os.FileMode(420), modTime: time.Unix(1454316838, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"tmpl/changelog.html": tmplChangelogHtml,
	"tmpl/index.html": tmplIndexHtml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"tmpl": &bintree{nil, map[string]*bintree{
		"changelog.html": &bintree{tmplChangelogHtml, map[string]*bintree{}},
		"index.html": &bintree{tmplIndexHtml, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

