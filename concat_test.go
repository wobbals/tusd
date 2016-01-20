package tusd_test

import (
	"net/http"
	"reflect"
	"testing"

	. "github.com/tus/tusd"
)

type concatPartialStore struct {
	t *testing.T
	zeroStore
}

func (s concatPartialStore) NewUpload(info FileInfo) (string, error) {
	if !info.IsPartial {
		s.t.Error("expected upload to be partial")
	}

	if info.IsFinal {
		s.t.Error("expected upload to not be final")
	}

	if len(info.PartialUploads) != 0 {
		s.t.Error("expected no partial uploads")
	}

	return "foo", nil
}

func (s concatPartialStore) GetInfo(id string) (FileInfo, error) {
	return FileInfo{
		IsPartial: true,
	}, nil
}

func (s concatPartialStore) ConcatUploads(id string, uploads []string) error {
	return nil
}

func TestConcatPartial(t *testing.T) {
	handler, _ := NewHandler(Config{
		MaxSize:  400,
		BasePath: "files",
		DataStore: concatPartialStore{
			t: t,
		},
	})

	(&httpTest{
		Name:   "Successful OPTIONS request",
		Method: "OPTIONS",
		URL:    "",
		ResHeader: map[string]string{
			"Tus-Extension": "creation,concatenation",
		},
		Code: http.StatusNoContent,
	}).Run(handler, t)

	(&httpTest{
		Name:   "Successful POST request",
		Method: "POST",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
			"Upload-Length": "300",
			"Upload-Concat": "partial",
		},
		Code: http.StatusCreated,
	}).Run(handler, t)

	(&httpTest{
		Name:   "Successful HEAD request",
		Method: "HEAD",
		URL:    "foo",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
		},
		Code: http.StatusNoContent,
		ResHeader: map[string]string{
			"Upload-Concat": "partial",
		},
	}).Run(handler, t)
}

type concatFinalStore struct {
	t *testing.T
	zeroStore
}

func (s concatFinalStore) NewUpload(info FileInfo) (string, error) {
	if info.IsPartial {
		s.t.Error("expected upload to not be partial")
	}

	if !info.IsFinal {
		s.t.Error("expected upload to be final")
	}

	if !reflect.DeepEqual(info.PartialUploads, []string{"a", "b"}) {
		s.t.Error("unexpected partial uploads")
	}

	return "foo", nil
}

func (s concatFinalStore) GetInfo(id string) (FileInfo, error) {
	if id == "a" || id == "b" {
		return FileInfo{
			IsPartial: true,
			Size:      5,
			Offset:    5,
		}, nil
	}

	if id == "c" {
		return FileInfo{
			IsPartial: true,
			Size:      5,
			Offset:    3,
		}, nil
	}

	if id == "foo" {
		return FileInfo{
			IsFinal:        true,
			PartialUploads: []string{"a", "b"},
			Size:           10,
			Offset:         10,
		}, nil
	}

	return FileInfo{}, ErrNotFound
}

func (s concatFinalStore) ConcatUploads(id string, uploads []string) error {
	if id != "foo" {
		s.t.Error("expected final file id to be foo")
	}

	if !reflect.DeepEqual(uploads, []string{"a", "b"}) {
		s.t.Errorf("expected Concatenating uploads to be a and b")
	}
	return nil
}

func TestConcatFinal(t *testing.T) {
	handler, _ := NewHandler(Config{
		MaxSize:  400,
		BasePath: "files",
		DataStore: concatFinalStore{
			t: t,
		},
	})

	(&httpTest{
		Name:   "Successful POST request",
		Method: "POST",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
			"Upload-Concat": "final; http://tus.io/files/a /files/b/",
		},
		Code: http.StatusCreated,
	}).Run(handler, t)

	(&httpTest{
		Name:   "Successful HEAD request",
		Method: "HEAD",
		URL:    "foo",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
		},
		Code: http.StatusNoContent,
		ResHeader: map[string]string{
			"Upload-Concat": "final; http://tus.io/files/a http://tus.io/files/b",
			"Upload-Length": "10",
			"Upload-Offset": "10",
		},
	}).Run(handler, t)

	(&httpTest{
		Name:   "Concatenating non finished upload (id: c)",
		Method: "POST",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
			"Upload-Concat": "final; http://tus.io/files/c",
		},
		Code: http.StatusBadRequest,
	}).Run(handler, t)

	handler, _ = NewHandler(Config{
		MaxSize:  9,
		BasePath: "files",
		DataStore: concatFinalStore{
			t: t,
		},
	})

	(&httpTest{
		Name:   "Exceeding MaxSize",
		Method: "POST",
		ReqHeader: map[string]string{
			"Tus-Resumable": "1.0.0",
			"Upload-Concat": "final; http://tus.io/files/a /files/b/",
		},
		Code: http.StatusRequestEntityTooLarge,
	}).Run(handler, t)
}
