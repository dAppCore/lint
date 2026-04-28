package io

import (
	stdio "io"
	"io/fs"
	"os"
	"path/filepath"

	. "dappco.re/go"
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
	AssertFalse(t, medium.Exists("missing.txt"))
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	got, err := medium.Read("alpha.txt")
	AssertNoError(t, err)
	AssertEqual(t, "alpha", got)
}

func TestIO_Medium_Read_Bad(t *T) {
	medium := ax7Medium(t)
	got, err := medium.Read("missing.txt")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

func TestIO_Medium_Read_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("empty.txt", ""))
	got, err := medium.Read("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

func TestIO_Medium_Write_Good(t *T) {
	medium := ax7Medium(t)
	err := medium.Write("nested/alpha.txt", "alpha")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile("nested/alpha.txt"))
}

func TestIO_Medium_Write_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Write("bad\x00path", "alpha")
	AssertError(t, err)
	AssertNotNil(t, medium)
}

func TestIO_Medium_Write_Ugly(t *T) {
	medium := ax7Medium(t)
	err := medium.Write("empty.txt", "")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile("empty.txt"))
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
	err := medium.WriteMode("bad\x00path", "alpha", 0o600)
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	got := medium.IsFile("alpha.txt")
	AssertTrue(t, got)
	AssertTrue(t, medium.Exists("alpha.txt"))
}

func TestIO_Medium_IsFile_Bad(t *T) {
	medium := ax7Medium(t)
	got := medium.IsFile("missing.txt")
	AssertFalse(t, got)
	AssertFalse(t, medium.Exists("missing.txt"))
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	err := medium.Delete("alpha.txt")
	AssertNoError(t, err)
	AssertFalse(t, medium.Exists("alpha.txt"))
}

func TestIO_Medium_Delete_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Delete("missing.txt")
	AssertError(t, err)
	AssertFalse(t, medium.Exists("missing.txt"))
}

func TestIO_Medium_Delete_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("nested/alpha.txt", "alpha"))
	err := medium.Delete("nested/alpha.txt")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsDir("nested"))
}

func TestIO_Medium_DeleteAll_Good(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("nested/alpha.txt", "alpha"))
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
	RequireNoError(t, medium.Write("old.txt", "alpha"))
	err := medium.Rename("old.txt", "new.txt")
	AssertNoError(t, err)
	AssertTrue(t, medium.IsFile("new.txt"))
}

func TestIO_Medium_Rename_Bad(t *T) {
	medium := ax7Medium(t)
	err := medium.Rename("missing.txt", "new.txt")
	AssertError(t, err)
	AssertFalse(t, medium.Exists("new.txt"))
}

func TestIO_Medium_Rename_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("old.txt", "alpha"))
	RequireNoError(t, medium.Write("new.txt", "beta"))
	err := medium.Rename("old.txt", "new.txt")
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	info, err := medium.Stat("alpha.txt")
	AssertNoError(t, err)
	AssertEqual(t, "alpha.txt", info.Name())
}

func TestIO_Medium_Stat_Bad(t *T) {
	medium := ax7Medium(t)
	info, err := medium.Stat("missing.txt")
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	file, err := medium.Open("alpha.txt")
	AssertNoError(t, err)
	AssertNotNil(t, file)
	ax7Close(t, file)
}

func TestIO_Medium_Open_Bad(t *T) {
	medium := ax7Medium(t)
	file, err := medium.Open("missing.txt")
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
	writer, err := medium.Create("bad\x00path")
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
	writer, err := medium.Append("bad\x00path")
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	reader, err := medium.ReadStream("alpha.txt")
	AssertNoError(t, err)
	data, readErr := stdio.ReadAll(reader)
	AssertNoError(t, readErr)
	ax7Close(t, reader)
	AssertEqual(t, "alpha", string(data))
}

func TestIO_Medium_ReadStream_Bad(t *T) {
	medium := ax7Medium(t)
	reader, err := medium.ReadStream("missing.txt")
	AssertError(t, err)
	AssertNil(t, reader)
}

func TestIO_Medium_ReadStream_Ugly(t *T) {
	medium := ax7Medium(t)
	RequireNoError(t, medium.Write("empty.txt", ""))
	reader, err := medium.ReadStream("empty.txt")
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
	writer, err := medium.WriteStream("bad\x00path")
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	got := medium.Exists("alpha.txt")
	AssertTrue(t, got)
	AssertTrue(t, medium.IsFile("alpha.txt"))
}

func TestIO_Medium_Exists_Bad(t *T) {
	medium := ax7Medium(t)
	got := medium.Exists("missing.txt")
	AssertFalse(t, got)
	AssertFalse(t, medium.IsFile("missing.txt"))
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
	RequireNoError(t, medium.Write("alpha.txt", "alpha"))
	got := medium.IsDir("alpha.txt")
	AssertFalse(t, got)
	AssertTrue(t, medium.IsFile("alpha.txt"))
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
