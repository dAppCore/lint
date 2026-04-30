package io

import (
	stdio "io"
	"io/fs"
	"os"
	"path/filepath"

	. "dappco.re/go"
)

const (
	ax7TestAlphaTxt10e34b       = "alpha.txt"
	ax7TestBadPatha1d16f        = "bad\x00path"
	ax7TestEmptyTxt20c33f       = "empty.txt"
	ax7TestMissingTxtacc363     = "missing.txt"
	ax7TestNestedAlphaTxt3ef370 = "nested/alpha.txt"
	ax7TestNewTxt5106a2         = "new.txt"
	ax7TestOldTxt8106ca         = "old.txt"
)

func ax7Medium(t *T) Medium {
	t.Helper()
	medium, err := NewSandboxed(t.TempDir())
	RequireNoError(t, err)
	return medium
}

func ax7Close(t *T, closer stdio.Closer) {
	t.Helper()
	RequireNoError(t, closer.Close())
}

func TestIO_NewSandboxed_Good(t *T) {
	medium, err := NewSandboxed(t.TempDir())
	AssertNoError(t, err)
	AssertNotNil(t, medium)
	AssertFalse(t, medium.Exists(ax7TestMissingTxtacc363))
}

func TestIO_NewSandboxed_Bad(t *T) {
	medium, err := NewSandboxed("")
	AssertNoError(t, err)
	AssertNotNil(t, medium)
	AssertTrue(t, medium.Exists("."))
}

func TestIO_NewSandboxed_Ugly(t *T) {
	root := filepath.Join(t.TempDir(), "nested", "..", "root")
	medium, err := NewSandboxed(root)
	AssertNoError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_Read_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	got, err := medium.Read(ax7TestAlphaTxt10e34b)
	AssertNoError(t, err)
	AssertEqual(t, "alpha", got)
}

func TestIO_Medium_Read_Bad(t *T) {
	medium := ax7Medium(t)
	got, err := medium.Read(ax7TestMissingTxtacc363)
	AssertError(t, err)
	AssertEqual(t, "", got)
}

func TestIO_Medium_Read_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestEmptyTxt20c33f, ""))
	got, err := medium.Read(ax7TestEmptyTxt20c33f)
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

func TestIO_Medium_Write_Good(t *T) {
	medium := ax7Medium(t)
	err := medium.Write(ax7TestNestedAlphaTxt3ef370, "alpha")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile(ax7TestNestedAlphaTxt3ef370))
}

func TestIO_Medium_Write_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Write(ax7TestBadPatha1d16f, "alpha")
	AssertError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_Write_Ugly(t *T) {
	medium := ax7Medium(t)
	err := medium.Write(ax7TestEmptyTxt20c33f, "")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile(ax7TestEmptyTxt20c33f))
}

func TestIO_Medium_WriteMode_Good(t *T) {
	medium := ax7Medium(t)
	err := medium.WriteMode("mode.txt", "alpha", 0o600)
	info, statErr := medium.Stat("mode.txt")
	AssertNoError(t, err)
	AssertNoError(t, statErr)
	AssertEqual(t, fs.FileMode(0o600), info.Mode().Perm())
}

func TestIO_Medium_WriteMode_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.WriteMode(ax7TestBadPatha1d16f, "alpha", 0o600)
	AssertError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_WriteMode_Ugly(t *T) {
	medium := ax7Medium(t)
	err := medium.WriteMode("nested/mode.txt", "", 0o640)
	info, statErr := medium.Stat("nested/mode.txt")
	AssertNoError(t, err)
	AssertNoError(t, statErr)
	AssertEqual(t, fs.FileMode(0o640), info.Mode().Perm())
}

func TestIO_Medium_EnsureDir_Good(t *T) {
	medium := ax7Medium(t)
	err := medium.EnsureDir("nested/dir")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsDir("nested/dir"))
}

func TestIO_Medium_EnsureDir_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.EnsureDir("bad\x00dir")
	AssertError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_EnsureDir_Ugly(t *T) {
	medium := ax7Medium(t)
	err := medium.EnsureDir("")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsDir("."))
}

func TestIO_Medium_IsFile_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	got := medium.IsFile(ax7TestAlphaTxt10e34b)
	AssertTrue(t, got)
	AssertTrue(t, medium.Exists(ax7TestAlphaTxt10e34b))
}

func TestIO_Medium_IsFile_Bad(t *T) {
	medium := ax7Medium(t)
	got := medium.IsFile(ax7TestMissingTxtacc363)
	AssertFalse(t, got)
	AssertFalse(t, medium.Exists(ax7TestMissingTxtacc363))
}

func TestIO_Medium_IsFile_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	AssertFalse(t, got)
	AssertTrue(t, medium.IsDir("dir"))
}

func TestIO_Medium_Delete_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	err := medium.Delete(ax7TestAlphaTxt10e34b)
	AssertNoError(t, err)
	AssertFalse(t, medium.Exists(ax7TestAlphaTxt10e34b))
}

func TestIO_Medium_Delete_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Delete(ax7TestMissingTxtacc363)
	AssertError(t, err)
	AssertFalse(t, medium.Exists(ax7TestMissingTxtacc363))
}

func TestIO_Medium_Delete_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestNestedAlphaTxt3ef370, "alpha"))
	err := medium.Delete(ax7TestNestedAlphaTxt3ef370)
	AssertNoError(t, err)
	AssertTrue(t, medium.IsDir("nested"))
}

func TestIO_Medium_DeleteAll_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestNestedAlphaTxt3ef370, "alpha"))
	err := medium.DeleteAll("nested")
	AssertNoError(t, err)
	AssertFalse(t, medium.Exists("nested"))
}

func TestIO_Medium_DeleteAll_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.DeleteAll("bad\x00dir")
	AssertError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_DeleteAll_Ugly(t *T) {
	medium := ax7Medium(t)
	err := medium.DeleteAll("missing")
	AssertNoError(t, err)
	AssertFalse(t, medium.Exists("missing"))
}

func TestIO_Medium_Rename_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestOldTxt8106ca, "alpha"))
	err := medium.Rename(ax7TestOldTxt8106ca, ax7TestNewTxt5106a2)
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile(ax7TestNewTxt5106a2))
}

func TestIO_Medium_Rename_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Rename(ax7TestMissingTxtacc363, ax7TestNewTxt5106a2)
	AssertError(t, err)
	AssertFalse(t, medium.Exists(ax7TestNewTxt5106a2))
}

func TestIO_Medium_Rename_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestOldTxt8106ca, "alpha"))
	RequireNoError(t, medium.Write(ax7TestNewTxt5106a2, "beta"))
	err := medium.Rename(ax7TestOldTxt8106ca, ax7TestNewTxt5106a2)
	AssertNoError(t, err)
}

func TestIO_Medium_List_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("dir/a.txt", "a"))
	RequireNoError(t, medium.Write("dir/b.txt", "b"))
	entries, err := medium.List("dir")
	AssertNoError(t, err)
	AssertLen(t, entries, 2)
}

func TestIO_Medium_List_Bad(t *T) {
	medium := ax7Medium(t)
	entries, err := medium.List("missing")
	AssertError(t, err)
	AssertNil(t, entries)
}

func TestIO_Medium_List_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.EnsureDir("empty"))
	entries, err := medium.List("empty")
	AssertNoError(t, err)
	AssertEmpty(t, entries)
}

func TestIO_Medium_Stat_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	info, err := medium.Stat(ax7TestAlphaTxt10e34b)
	AssertNoError(t, err)
	AssertEqual(t, ax7TestAlphaTxt10e34b, info.Name())
}

func TestIO_Medium_Stat_Bad(t *T) {
	medium := ax7Medium(t)
	info, err := medium.Stat(ax7TestMissingTxtacc363)
	AssertError(t, err)
	AssertNil(t, info)
}

func TestIO_Medium_Stat_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	AssertNoError(t, err)
	AssertTrue(t, info.IsDir())
}

func TestIO_Medium_Open_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	file, err := medium.Open(ax7TestAlphaTxt10e34b)
	AssertNoError(t, err)
	AssertNotNil(t, file)
	ax7Close(t, file)
}

func TestIO_Medium_Open_Bad(t *T) {
	medium := ax7Medium(t)
	file, err := medium.Open(ax7TestMissingTxtacc363)
	AssertError(t, err)
	AssertNil(t, file)
}

func TestIO_Medium_Open_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.EnsureDir("dir"))
	file, err := medium.Open("dir")
	AssertNoError(t, err)
	ax7Close(t, file)
}

func TestIO_Medium_Create_Good(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.Create("created.txt")
	AssertNoError(t, err)
	_, writeErr := writer.Write([]byte("alpha"))
	AssertNoError(t, writeErr)
	ax7Close(t, writer)
}

func TestIO_Medium_Create_Bad(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.Create(ax7TestBadPatha1d16f)
	AssertError(t, err)
	AssertNil(t, writer)
}

func TestIO_Medium_Create_Ugly(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.Create("nested/created.txt")
	AssertNoError(t, err)
	ax7Close(t, writer)
	AssertTrue(t, medium.IsFile("nested/created.txt"))
}

func TestIO_Medium_Append_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	AssertNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	AssertNoError(t, writeErr)
	ax7Close(t, writer)
}

func TestIO_Medium_Append_Bad(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.Append(ax7TestBadPatha1d16f)
	AssertError(t, err)
	AssertNil(t, writer)
}

func TestIO_Medium_Append_Ugly(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.Append("created-by-append.txt")
	AssertNoError(t, err)
	ax7Close(t, writer)
	AssertTrue(t, medium.IsFile("created-by-append.txt"))
}

func TestIO_Medium_ReadStream_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	reader, err := medium.ReadStream(ax7TestAlphaTxt10e34b)
	AssertNoError(t, err)
	data, readErr := stdio.ReadAll(reader)
	AssertNoError(t, readErr)
	ax7Close(t, reader)
	AssertEqual(t, "alpha", string(data))
}

func TestIO_Medium_ReadStream_Bad(t *T) {
	medium := ax7Medium(t)
	reader, err := medium.ReadStream(ax7TestMissingTxtacc363)
	AssertError(t, err)
	AssertNil(t, reader)
}

func TestIO_Medium_ReadStream_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestEmptyTxt20c33f, ""))
	reader, err := medium.ReadStream(ax7TestEmptyTxt20c33f)
	AssertNoError(t, err)
	data, readErr := stdio.ReadAll(reader)
	AssertNoError(t, readErr)
	ax7Close(t, reader)
	AssertEqual(t, "", string(data))
}

func TestIO_Medium_WriteStream_Good(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.WriteStream("stream.txt")
	AssertNoError(t, err)
	_, writeErr := writer.Write([]byte("alpha"))
	AssertNoError(t, writeErr)
	ax7Close(t, writer)
}

func TestIO_Medium_WriteStream_Bad(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.WriteStream(ax7TestBadPatha1d16f)
	AssertError(t, err)
	AssertNil(t, writer)
}

func TestIO_Medium_WriteStream_Ugly(t *T) {
	medium := ax7Medium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	AssertNoError(t, err)
	ax7Close(t, writer)
	AssertTrue(t, medium.IsFile("nested/stream.txt"))
}

func TestIO_Medium_Exists_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	got := medium.Exists(ax7TestAlphaTxt10e34b)
	AssertTrue(t, got)
	AssertTrue(t, medium.IsFile(ax7TestAlphaTxt10e34b))
}

func TestIO_Medium_Exists_Bad(t *T) {
	medium := ax7Medium(t)
	got := medium.Exists(ax7TestMissingTxtacc363)
	AssertFalse(t, got)
	AssertFalse(t, medium.IsFile(ax7TestMissingTxtacc363))
}

func TestIO_Medium_Exists_Ugly(t *T) {
	medium := ax7Medium(t)
	got := medium.Exists(".")
	AssertTrue(t, got)
	AssertTrue(t, medium.IsDir("."))
}

func TestIO_Medium_IsDir_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	AssertTrue(t, got)
	AssertTrue(t, medium.Exists("dir"))
}

func TestIO_Medium_IsDir_Bad(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write(ax7TestAlphaTxt10e34b, "alpha"))
	got := medium.IsDir(ax7TestAlphaTxt10e34b)
	AssertFalse(t, got)
	AssertTrue(t, medium.IsFile(ax7TestAlphaTxt10e34b))
}

func TestIO_Medium_IsDir_Ugly(t *T) {
	medium := ax7Medium(t)
	got := medium.IsDir("missing")
	AssertFalse(t, got)
	AssertFalse(t, medium.Exists("missing"))
}

func TestIO_Medium_Local_Good(t *T) {
	path := filepath.Join(t.TempDir(), "local.txt")
	RequireNoError(t, os.WriteFile(path, []byte("alpha"), 0o644))
	got, err := Local.Read(path)
	AssertNoError(t, err)
	AssertEqual(t, "alpha", got)
}
