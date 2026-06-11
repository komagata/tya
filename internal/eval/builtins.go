package eval

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mathpkg "math"
	mathrand "math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func installBuiltins(env *Env, in io.Reader, out io.Writer, processArgs []string) {
	var lineReader *bufio.Reader
	if in != nil {
		lineReader = bufio.NewReader(in)
	}
	for _, name := range []string{"Number", "String", "Bytes", "Array", "Dict", "Boolean", "Nil"} {
		env.set(name, primitiveClass(name))
	}
	env.set("print", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("print expects 1 argument")
		}
		text, err := displayString(args[0], env)
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(out, text)
		return nil, nil
	}))
	env.set("println", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("println expects 1 argument")
		}
		text, err := displayString(args[0], env)
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(out, text)
		return nil, nil
	}))
	env.set("to_string", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_string expects 1 argument")
		}
		return displayString(args[0], env)
	}))
	env.set("inspect", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("inspect expects 1 argument")
		}
		return inspectString(args[0], env)
	}))
	env.set("read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("read_line expects 0 arguments")
		}
		if lineReader == nil {
			return nil, nil
		}
		line, err := lineReader.ReadString('\n')
		if errors.Is(err, io.EOF) && line == "" {
			return nil, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
	}))
	env.set("delete", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("delete expects 2 arguments")
		}
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("delete expects dictionary")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("delete expects string key")
		}
		delete(obj, key)
		return nil, nil
	}))
	env.set("equal", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("equal expects 2 arguments")
		}
		return deepEqual(args[0], args[1]), nil
	}))
	env.set("regex_compile", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("regex_compile expects 2 arguments")
		}
		pattern, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.compile: pattern must be a string", "invalid_pattern_kind")
		}
		options, ok := args[1].(Dict)
		if !ok {
			return nil, regexRaised("regex.compile: options must be a dictionary", "invalid_options")
		}
		compiled, err := compileRegexValue(pattern, options)
		if err != nil {
			return nil, err
		}
		return regexObject(compiled), nil
	}))
	env.set("ord", Builtin(func(args []Value) (Value, error) {
		s, err := oneString("ord", args)
		if err != nil {
			return nil, err
		}
		if len(s) == 0 {
			return nil, &raisedSignal{value: "ord: argument must be a non-empty string"}
		}
		r, _ := utf8.DecodeRuneInString(s)
		return int64(r), nil
	}))
	env.set("chr", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("chr expects 1 argument")
		}
		v, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("chr expects int argument")
		}
		if v < 0 || v > utf8.MaxRune || (v >= 0xD800 && v <= 0xDFFF) {
			return nil, &raisedSignal{value: "chr: code point out of range"}
		}
		return string(rune(v)), nil
	}))
	env.set("byte_len", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("byte_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len(text)), nil
	}))
	env.set("char_len", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("char_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len([]rune(text))), nil
	}))
	env.set("read_file", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("read_file", args)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if !utf8.Valid(data) {
			return nil, &raisedSignal{value: "read_file: invalid UTF-8"}
		}
		return string(data), nil
	}))
	env.set("write_file", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("write_file expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("write_file expects string path")
		}
		text, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("write_file expects string text")
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			return nil, err
		}
		return nil, nil
	}))
	env.set("file_exists", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_exists", args)
		if err != nil {
			return nil, err
		}
		_, statErr := os.Stat(path)
		if statErr == nil {
			return true, nil
		}
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		return nil, statErr
	}))
	env.set("dir_list", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_list", args)
		if err != nil {
			return nil, err
		}
		entries, listErr := os.ReadDir(path)
		if listErr != nil {
			return nil, &raisedSignal{value: listErr.Error()}
		}
		arr := &Array{items: make([]Value, 0, len(entries))}
		for _, e := range entries {
			arr.items = append(arr.items, e.Name())
		}
		return arr, nil
	}))
	env.set("dir_mkdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_mkdir", args)
		if err != nil {
			return nil, err
		}
		if mkErr := os.Mkdir(path, 0755); mkErr != nil {
			return nil, &raisedSignal{value: mkErr.Error()}
		}
		return nil, nil
	}))
	env.set("dir_rmdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_rmdir", args)
		if err != nil {
			return nil, err
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return nil, &raisedSignal{value: rmErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_remove", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_remove", args)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, &raisedSignal{value: statErr.Error()}
		}
		if info.IsDir() {
			return nil, &raisedSignal{value: "file.remove: target is a directory"}
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return nil, &raisedSignal{value: rmErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_rename", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_rename expects 2 arguments")
		}
		oldPath, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("file_rename expects string old path")
		}
		newPath, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("file_rename expects string new path")
		}
		if rnErr := os.Rename(oldPath, newPath); rnErr != nil {
			return nil, &raisedSignal{value: rnErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_stat", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_stat", args)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, &raisedSignal{value: statErr.Error()}
		}
		kind := "other"
		if info.Mode().IsRegular() {
			kind = "file"
		} else if info.IsDir() {
			kind = "dir"
		}
		out := Dict{}
		out["kind"] = kind
		out["size"] = int64(info.Size())
		mode := info.Mode()
		out["readable"] = mode&0444 != 0
		out["writable"] = mode&0222 != 0
		out["executable"] = mode&0111 != 0
		out["mode"] = int64(mode.Perm())
		return out, nil
	}))
	env.set("dir_mkdir_all", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_mkdir_all", args)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, &raisedSignal{value: "filesystem.mkdir_all: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("dir_remove_all", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_remove_all", args)
		if err != nil {
			return nil, err
		}
		if filesystemDangerousPath(path) {
			return nil, &raisedSignal{value: "filesystem.remove_all: dangerous path"}
		}
		if err := os.RemoveAll(path); err != nil {
			return nil, &raisedSignal{value: "filesystem.remove_all: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("dir_temp_dir", Builtin(func(args []Value) (Value, error) {
		prefix := "tya"
		if len(args) > 1 {
			return nil, fmt.Errorf("dir_temp_dir expects 0 or 1 arguments")
		}
		if len(args) == 1 {
			var ok bool
			prefix, ok = args[0].(string)
			if !ok {
				return nil, &raisedSignal{value: "filesystem.temp_dir: prefix must be string"}
			}
		}
		path, err := os.MkdirTemp("", prefix)
		if err != nil {
			return nil, &raisedSignal{value: "filesystem.temp_dir: " + err.Error()}
		}
		return path, nil
	}))
	env.set("dir_walk", Builtin(func(args []Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, fmt.Errorf("dir_walk expects 2 or 3 arguments")
		}
		root, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "filesystem.walk: path must be string"}
		}
		fn, ok := args[1].(*Function)
		if !ok {
			return nil, &raisedSignal{value: "filesystem.walk: callback must be function"}
		}
		includeDirs, includeFiles := true, true
		if len(args) == 3 && args[2] != nil {
			opts, ok := args[2].(Dict)
			if !ok {
				return nil, &raisedSignal{value: "filesystem.walk: options must be dictionary"}
			}
			if v, has := opts["include_dirs"].(bool); has {
				includeDirs = v
			}
			if v, has := opts["include_files"].(bool); has {
				includeFiles = v
			}
		}
		var paths []string
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != root {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
			return nil, &raisedSignal{value: "filesystem.walk: " + err.Error()}
		}
		sort.Strings(paths)
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				return nil, &raisedSignal{value: "filesystem.walk: " + err.Error()}
			}
			isDir := info.IsDir()
			if (isDir && !includeDirs) || (!isDir && !includeFiles) {
				continue
			}
			kind := "file"
			if isDir {
				kind = "dir"
			}
			entry := Dict{"path": path, "name": filepath.Base(path), "kind": kind, "stat": fileInfoDict(info)}
			if _, err := callValue(fn, []Value{entry}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}))
	env.set("path_expand_user", Builtin(func(args []Value) (Value, error) {
		v, err := oneString("path_expand_user", args)
		if err != nil {
			return nil, err
		}
		if v == "" || v[0] != '~' {
			return v, nil
		}
		home := os.Getenv("HOME")
		if v == "~" {
			return home, nil
		}
		if len(v) > 1 && v[1] == '/' {
			return home + v[1:], nil
		}
		return v, nil
	}))
	env.set("cwd", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("cwd expects 0 arguments")
		}
		dir, cwdErr := os.Getwd()
		if cwdErr != nil {
			return nil, &raisedSignal{value: cwdErr.Error()}
		}
		return dir, nil
	}))
	env.set("chdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("chdir", args)
		if err != nil {
			return nil, err
		}
		if cdErr := os.Chdir(path); cdErr != nil {
			return nil, &raisedSignal{value: cdErr.Error()}
		}
		return nil, nil
	}))
	registerV24Builtins(env)
	env.set("args", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("args expects 0 arguments")
		}
		arr := &Array{items: make([]Value, 0, len(processArgs))}
		for _, arg := range processArgs {
			arr.items = append(arr.items, arg)
		}
		return arr, nil
	}))
	env.set("env", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("env expects 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("env expects string name")
		}
		value, ok := os.LookupEnv(name)
		if !ok {
			return nil, nil
		}
		return value, nil
	}))
	env.set("environ", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("environ expects 0 arguments")
		}
		out := Dict{}
		for _, item := range os.Environ() {
			key, value, ok := strings.Cut(item, "=")
			if ok {
				out[key] = value
			}
		}
		return out, nil
	}))
	env.set("setenv", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("setenv expects 2 arguments")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("setenv expects string name")
		}
		value, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("setenv expects string value")
		}
		if strings.ContainsRune(name, 0) || strings.ContainsRune(value, 0) {
			return nil, &raisedSignal{value: "os.env: NUL byte not allowed"}
		}
		if err := os.Setenv(name, value); err != nil {
			return nil, &raisedSignal{value: "os.env: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("unsetenv", Builtin(func(args []Value) (Value, error) {
		name, err := oneString("unsetenv", args)
		if err != nil {
			return nil, err
		}
		if strings.ContainsRune(name, 0) {
			return nil, &raisedSignal{value: "os.env: NUL byte not allowed"}
		}
		if err := os.Unsetenv(name); err != nil {
			return nil, &raisedSignal{value: "os.env: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("exit", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("exit expects 1 argument")
		}
		code, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("exit expects int code")
		}
		return nil, &ExitError{Code: int(code)}
	}))
	env.set("panic", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("panic expects 1 argument")
		}
		return nil, fmt.Errorf("panic: %s", stringify(args[0]))
	}))
	env.set("error", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("error expects 1 or 2 arguments")
		}
		if message, ok := args[0].(string); ok {
			errValue := &ErrorValue{Message: message, Kind: "error", Code: "", Data: Dict{}, Cause: nil}
			if len(args) == 2 {
				options, ok := args[1].(Dict)
				if !ok {
					return nil, fmt.Errorf("error options must be a dictionary")
				}
				if err := applyErrorOptions(errValue, options); err != nil {
					return nil, err
				}
			}
			return errValue, nil
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("error message must be string")
		}
		options, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("error expects string message or dictionary options")
		}
		message, ok := options["message"].(string)
		if !ok {
			return nil, fmt.Errorf("error options message must be string")
		}
		errValue := &ErrorValue{Message: message, Kind: "error", Code: "", Data: Dict{}, Cause: nil}
		if err := applyErrorOptions(errValue, options); err != nil {
			return nil, err
		}
		return errValue, nil
	}))
	env.set("div", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("div expects 2 arguments")
		}
		left, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("div expects int left operand")
		}
		right, ok := args[1].(int64)
		if !ok {
			return nil, fmt.Errorf("div expects int right operand")
		}
		if right == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}))
	env.set("to_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_int expects 1 argument")
		}
		return parseIntValue(args[0])
	}))
	env.set("to_float", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_float expects 1 argument")
		}
		return parseFloatValue(args[0])
	}))
}

func registerV24Builtins(env *Env) {
	env.set("time_now", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time_now expects 0 arguments")
		}
		return timeObject(float64(time.Now().UnixNano())/1e9, false, false), nil
	}))
	env.set("time_monotonic", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time_monotonic expects 0 arguments")
		}
		return timeObject(time.Since(tyaMonotonicStart).Seconds(), true, false), nil
	}))
	env.set("time_unix", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("time_unix expects 2 arguments")
		}
		secs, ok := numberAsFloat(args[0])
		if !ok {
			return nil, timeRaised("time.unix: seconds must be a number", "invalid_seconds")
		}
		nanos, ok := numberAsInt(args[1])
		if !ok {
			return nil, timeRaised("time.unix: nanos must be an integer", "invalid_nanos")
		}
		return timeObject(secs+float64(nanos)/1e9, false, false), nil
	}))
	env.set("time_duration", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("time_duration expects 2 arguments")
		}
		secs, ok := numberAsFloat(args[0])
		if !ok {
			return nil, timeRaised("time.duration: seconds must be a number", "invalid_seconds")
		}
		options, ok := args[1].(Dict)
		if !ok {
			return nil, timeRaised("time.duration: options must be a dictionary", "invalid_options")
		}
		for key, value := range options {
			n, ok := numberAsFloat(value)
			if !ok {
				return nil, timeRaised("time.duration: option "+key+" must be a number", "invalid_option")
			}
			switch key {
			case "minutes":
				secs += n * 60
			case "hours":
				secs += n * 3600
			case "milliseconds":
				secs += n / 1e3
			case "microseconds":
				secs += n / 1e6
			case "nanoseconds":
				secs += n / 1e9
			default:
				return nil, timeRaised("time.duration: unknown option "+key, "unknown_option")
			}
		}
		return durationObject(secs), nil
	}))
	env.set("time_sleep", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time_sleep expects 1 argument")
		}
		secs, ok := durationSeconds(args[0])
		if !ok {
			return nil, timeRaised("time.sleep: argument must be a duration or number", "invalid_duration")
		}
		if secs < 0 {
			return nil, timeRaised("time.sleep: negative duration", "negative_duration")
		}
		time.Sleep(time.Duration(secs * float64(time.Second)))
		return nil, nil
	}))
	env.set("time_format", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("time_format expects 1 or 2 arguments")
		}
		secs, monotonic, ok := timeSeconds(args[0])
		if !ok {
			return nil, timeRaised("time.format: argument must be a time", "invalid_time")
		}
		local := false
		if dict, ok := args[0].(Dict); ok {
			local, _ = dict["__time_local"].(bool)
		}
		layout := "rfc3339"
		if len(args) == 2 {
			s, ok := args[1].(string)
			if !ok {
				return nil, timeRaised("time.format: layout must be a string", "invalid_layout")
			}
			layout = s
		}
		return formatTimeValue(secs, monotonic, local, layout)
	}))
	env.set("time_parse", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("time_parse expects 1 or 2 arguments")
		}
		s, ok := args[0].(string)
		if !ok {
			return nil, timeRaised("time.parse: text must be a string", "invalid_text")
		}
		layout := "rfc3339"
		if len(args) == 2 {
			var ok bool
			layout, ok = args[1].(string)
			if !ok {
				return nil, timeRaised("time.parse: layout must be a string", "invalid_layout")
			}
		}
		switch layout {
		case "rfc3339", "iso":
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return nil, timeRaised("time.parse: invalid timestamp", "invalid_timestamp")
			}
			return timeObject(float64(t.UnixNano())/1e9, false, false), nil
		case "date":
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return nil, timeRaised("time.parse: invalid timestamp", "invalid_timestamp")
			}
			return timeObject(float64(t.UnixNano())/1e9, false, false), nil
		case "unix":
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, timeRaised("time.parse: invalid timestamp", "invalid_timestamp")
			}
			return timeObject(float64(n), false, false), nil
		}
		return nil, timeRaised("time.parse: unknown layout", "unknown_layout")
	}))
	env.set("time_since", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time_since expects 1 argument")
		}
		t, _, ok := timeSeconds(args[0])
		if !ok {
			return nil, timeRaised("time.since: argument must be a time", "invalid_time")
		}
		return durationObject(float64(time.Now().UnixNano())/1e9 - t), nil
	}))

	env.set("random_seed", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("random_seed expects 1 argument")
		}
		var seed int64
		switch v := args[0].(type) {
		case int64:
			seed = v
		case float64:
			seed = int64(v)
		case string:
			h := uint64(14695981039346656037)
			for _, b := range []byte(v) {
				h ^= uint64(b)
				h *= 1099511628211
			}
			seed = int64(h)
		default:
			return nil, fmt.Errorf("random_seed expects int or string")
		}
		tyaRng = mathrand.New(mathrand.NewSource(seed))
		return nil, nil
	}))
	env.set("random_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("random_int expects 2 arguments")
		}
		mn, ok1 := numberAsInt(args[0])
		mx, ok2 := numberAsInt(args[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("random_int expects ints")
		}
		if mx < mn {
			return nil, &raisedSignal{value: "random.int: max < min"}
		}
		return mn + tyaRng.Int63n(mx-mn+1), nil
	}))
	env.set("random_float", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("random_float expects 0 arguments")
		}
		return tyaRng.Float64(), nil
	}))

	addMath := func(name string, fn func(float64) float64) {
		env.set(name, Builtin(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("%s expects 1 argument", name)
			}
			x, ok := numberAsFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("%s expects number", name)
			}
			return fn(x), nil
		}))
	}
	env.set("math_sqrt", Builtin(func(args []Value) (Value, error) {
		x, _ := numberAsFloat(args[0])
		if x < 0 {
			return nil, &raisedSignal{value: "math.sqrt: negative argument"}
		}
		return mathpkg.Sqrt(x), nil
	}))
	env.set("math_pow", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("math_pow expects 2 arguments")
		}
		x, _ := numberAsFloat(args[0])
		y, _ := numberAsFloat(args[1])
		return mathpkg.Pow(x, y), nil
	}))
	addMath("math_floor", mathpkg.Floor)
	addMath("math_ceil", mathpkg.Ceil)
	env.set("math_round", Builtin(func(args []Value) (Value, error) {
		x, _ := numberAsFloat(args[0])
		if x >= 0 {
			return mathpkg.Floor(x + 0.5), nil
		}
		return -mathpkg.Floor(-x + 0.5), nil
	}))
	addMath("math_trunc", mathpkg.Trunc)
	addLog := func(name string, fn func(float64) float64) {
		env.set(name, Builtin(func(args []Value) (Value, error) {
			x, _ := numberAsFloat(args[0])
			if x <= 0 {
				return nil, &raisedSignal{value: "math: non-positive argument to log"}
			}
			return fn(x), nil
		}))
	}
	addLog("math_log", mathpkg.Log)
	addLog("math_log2", mathpkg.Log2)
	addLog("math_log10", mathpkg.Log10)
	addMath("math_exp", mathpkg.Exp)
	addMath("math_sin", mathpkg.Sin)
	addMath("math_cos", mathpkg.Cos)
	addMath("math_tan", mathpkg.Tan)
	addMath("math_asin", mathpkg.Asin)
	addMath("math_acos", mathpkg.Acos)
	addMath("math_atan", mathpkg.Atan)
	env.set("math_atan2", Builtin(func(args []Value) (Value, error) {
		y, _ := numberAsFloat(args[0])
		x, _ := numberAsFloat(args[1])
		return mathpkg.Atan2(y, x), nil
	}))

	env.set("process_run", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("process_run expects 1 or 2 arguments")
		}
		opts := Dict{}
		if len(args) == 2 && args[1] != nil {
			var ok bool
			opts, ok = args[1].(Dict)
			if !ok {
				return nil, &raisedSignal{value: "process.run: options must be a dictionary"}
			}
		}
		allowed := map[string]bool{"cwd": true, "env": true, "clear_env": true, "stdin": true, "input": true, "capture_stdout": true, "capture_stderr": true, "timeout": true, "shell": true}
		for key := range opts {
			if !allowed[key] {
				return nil, &raisedSignal{value: "process.run: unknown option " + key}
			}
		}
		shell := false
		if v, has := opts["shell"]; has {
			var ok bool
			shell, ok = v.(bool)
			if !ok {
				return nil, &raisedSignal{value: "process.run: shell must be bool"}
			}
		}
		var cmdArgs []string
		switch command := args[0].(type) {
		case string:
			if command == "" {
				return nil, &raisedSignal{value: "process.run: command must be non-empty"}
			}
			if !shell {
				return nil, &raisedSignal{value: "process.run: string command requires shell option"}
			}
			cmdArgs = []string{"sh", "-c", command}
		case *Array:
			if len(command.items) == 0 {
				return nil, &raisedSignal{value: "process.run: command must be a non-empty array"}
			}
			cmdArgs = make([]string, len(command.items))
			for i, v := range command.items {
				s, ok := v.(string)
				if !ok {
					return nil, &raisedSignal{value: "process.run: command items must be strings"}
				}
				cmdArgs[i] = s
			}
		default:
			return nil, &raisedSignal{value: "process.run: command must be a string or array"}
		}
		var ctx context.Context = context.Background()
		var cancel context.CancelFunc
		if v, has := opts["timeout"]; has {
			secs, ok := numberAsFloat(v)
			if !ok {
				return nil, &raisedSignal{value: "process.run: timeout must be a number"}
			}
			ctx, cancel = context.WithTimeout(ctx, time.Duration(secs*float64(time.Second)))
			defer cancel()
		}
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		if cwd, has := opts["cwd"].(string); has {
			cmd.Dir = cwd
		}
		input := opts["stdin"]
		if input == nil {
			input = opts["input"]
		}
		if input != nil {
			switch v := input.(type) {
			case string:
				cmd.Stdin = strings.NewReader(v)
			case *Bytes:
				cmd.Stdin = bytes.NewReader(v.data)
			default:
				return nil, &raisedSignal{value: "process.run: stdin must be string or bytes"}
			}
		}
		if clear, has := opts["clear_env"].(bool); has && clear {
			cmd.Env = []string{}
		} else {
			cmd.Env = os.Environ()
		}
		if envDict, has := opts["env"].(Dict); has {
			for k, v := range envDict {
				s, ok := v.(string)
				if !ok {
					return nil, &raisedSignal{value: "process.run: env values must be strings"}
				}
				cmd.Env = append(cmd.Env, k+"="+s)
			}
		}
		err := cmd.Run()
		exitCode := 0
		timedOut := ctx.Err() == context.DeadlineExceeded
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, &raisedSignal{value: "process.run: " + err.Error()}
			}
		}
		out := Dict{}
		out["status"] = int64(exitCode)
		out["exit_code"] = int64(exitCode)
		out["success"] = exitCode == 0
		out["stdout"] = stdoutBuf.String()
		out["stderr"] = stderrBuf.String()
		out["timed_out"] = timedOut
		return out, nil
	}))
	env.set("process_exec", Builtin(func(args []Value) (Value, error) {
		return nil, &raisedSignal{value: "process.exec: unsupported on this runtime"}
	}))

	digestInput := func(name string, args []Value) ([]byte, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s expects 1 argument", name)
		}
		switch v := args[0].(type) {
		case string:
			return []byte(v), nil
		case *Bytes:
			return v.data, nil
		}
		return nil, &raisedSignal{value: name + ": argument must be a string or bytes"}
	}
	env.set("digest_md5", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.md5", args)
		if err != nil {
			return nil, err
		}
		h := md5.Sum(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha1", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha1", args)
		if err != nil {
			return nil, err
		}
		h := sha1.Sum(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha256", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha256", args)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha384", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha384", args)
		if err != nil {
			return nil, err
		}
		h := sha512.Sum384(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha512", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha512", args)
		if err != nil {
			return nil, err
		}
		h := sha512.Sum512(data)
		return hex.EncodeToString(h[:]), nil
	}))

	env.set("secure_random_bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("secure_random_bytes expects 1 argument")
		}
		n, ok := numberAsInt(args[0])
		if !ok || n < 0 || n > 4096 {
			return nil, &raisedSignal{value: "secure_random.bytes: count out of range"}
		}
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return nil, &raisedSignal{value: "secure_random: entropy unavailable"}
		}
		return &Bytes{data: buf}, nil
	}))
	env.set("secure_random_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("secure_random_int expects 2 arguments")
		}
		mn, ok1 := numberAsInt(args[0])
		mx, ok2 := numberAsInt(args[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("secure_random_int expects ints")
		}
		if mx < mn {
			return nil, &raisedSignal{value: "secure_random.int: max < min"}
		}
		rng := uint64(mx - mn + 1)
		threshold := uint64(0)
		if rng != 0 {
			threshold = (^uint64(0) - rng + 1) % rng
		}
		for {
			var b [8]byte
			if _, err := rand.Read(b[:]); err != nil {
				return nil, &raisedSignal{value: "secure_random.int: entropy unavailable"}
			}
			r := binary.BigEndian.Uint64(b[:])
			if r >= threshold {
				return mn + int64(r%rng), nil
			}
		}
	}))
	registerV25Builtins(env)
	registerV41Builtins(env)
}

func registerV41Builtins(env *Env) {
	// runtime_gc_stats() — eval has no real GC; return zeros for parity with C runtime.
	env.set("runtime_gc_stats", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("runtime_gc_stats expects 0 arguments")
		}
		out := Dict{
			"alloc_count":   float64(0),
			"alloc_bytes":   float64(0),
			"freed_count":   float64(0),
			"freed_bytes":   float64(0),
			"live_count":    float64(0),
			"live_bytes":    float64(0),
			"collect_count": float64(0),
			"threshold":     float64(0),
		}
		return out, nil
	}))
	// runtime_gc_collect() — eval has no real GC; no-op for parity.
	env.set("runtime_gc_collect", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("runtime_gc_collect expects 0 arguments")
		}
		return nil, nil
	}))
	// v0.42 channel: eval is single-threaded; the stubs below raise so
	// programs that require concurrency are routed through the C runtime.
	stub := func(name string, want int) Builtin {
		return func(args []Value) (Value, error) {
			if len(args) != want {
				return nil, fmt.Errorf("%s expects %d arguments", name, want)
			}
			return nil, &raisedSignal{value: name + ": eval interpreter does not support concurrency; use tya run"}
		}
	}
	env.set("channel_new", stub("channel_new", 1))
	env.set("channel_send", stub("channel_send", 2))
	env.set("channel_receive", stub("channel_receive", 1))
	env.set("channel_receive_timeout", stub("channel_receive_timeout", 2))
	env.set("channel_close", stub("channel_close", 1))
	env.set("channel_closed_p", stub("channel_closed_p", 1))
	env.set("channel_select", stub("channel_select", 1))
	env.set("task_cancel", stub("task_cancel", 1))
	env.set("task_is_cancelled_p", stub("task_is_cancelled_p", 1))
	env.set("task_current", stub("task_current", 0))
	env.set("sync_mutex_new", stub("sync_mutex_new", 0))
	env.set("sync_lock", stub("sync_lock", 1))
	env.set("sync_unlock", stub("sync_unlock", 1))
	env.set("sync_atomic_integer_new", stub("sync_atomic_integer_new", 1))
	env.set("sync_atomic_integer_add", stub("sync_atomic_integer_add", 2))
	env.set("sync_atomic_integer_load", stub("sync_atomic_integer_load", 1))
	env.set("sync_atomic_integer_store", stub("sync_atomic_integer_store", 2))
	env.set("sync_atomic_integer_cas", stub("sync_atomic_integer_cas", 3))
	env.set("sync_wait_group_new", stub("sync_wait_group_new", 0))
	env.set("sync_wait_group_add", stub("sync_wait_group_add", 2))
	env.set("sync_wait_group_done", stub("sync_wait_group_done", 1))
	env.set("sync_wait_group_wait", stub("sync_wait_group_wait", 1))
}

func registerV25Builtins(env *Env) {
	env.set("bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes expects 1 argument")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, &raisedSignal{value: "bytes: argument must be an array of ints"}
		}
		out := make([]byte, len(arr.items))
		for i, item := range arr.items {
			n, ok := numberAsInt(item)
			if !ok {
				return nil, &raisedSignal{value: "bytes: items must be ints"}
			}
			if n < 0 || n > 255 {
				return nil, &raisedSignal{value: "bytes: item out of 0..255"}
			}
			out[i] = byte(n)
		}
		return &Bytes{data: out}, nil
	}))
	env.set("bytes_of", Builtin(func(args []Value) (Value, error) {
		s, err := oneString("bytes_of", args)
		if err != nil {
			return nil, err
		}
		return &Bytes{data: []byte(s)}, nil
	}))
	env.set("bytes_text", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes_text expects 1 argument")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_text: argument must be bytes"}
		}
		for _, c := range b.data {
			if c == 0 {
				return nil, &raisedSignal{value: "bytes_text: NUL byte not allowed"}
			}
		}
		if !utf8.Valid(b.data) {
			return nil, &raisedSignal{value: "bytes_text: invalid UTF-8"}
		}
		return string(b.data), nil
	}))
	env.set("bytes_array", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes_array expects 1 argument")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_array: argument must be bytes"}
		}
		out := &Array{items: make([]Value, len(b.data))}
		for i, c := range b.data {
			out.items[i] = int64(c)
		}
		return out, nil
	}))
	env.set("bytes_concat", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("bytes_concat expects 2 arguments")
		}
		a, aok := args[0].(*Bytes)
		b, bok := args[1].(*Bytes)
		if !aok || !bok {
			return nil, &raisedSignal{value: "bytes_concat: arguments must be bytes"}
		}
		out := make([]byte, len(a.data)+len(b.data))
		copy(out, a.data)
		copy(out[len(a.data):], b.data)
		return &Bytes{data: out}, nil
	}))
	env.set("bytes_slice", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("bytes_slice expects 3 arguments")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_slice: first argument must be bytes"}
		}
		s, sok := numberAsInt(args[1])
		e, eok := numberAsInt(args[2])
		if !sok || !eok {
			return nil, &raisedSignal{value: "bytes_slice: indices must be ints"}
		}
		if s < 0 || e < s || int(e) > len(b.data) {
			return nil, &raisedSignal{value: "bytes_slice: index out of range"}
		}
		return &Bytes{data: append([]byte{}, b.data[s:e]...)}, nil
	}))
	env.set("file_read_bytes", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_read_bytes", args)
		if err != nil {
			return nil, err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, &raisedSignal{value: rerr.Error()}
		}
		return &Bytes{data: data}, nil
	}))
	env.set("file_write_bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_write_bytes expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.write_bytes: path must be a string"}
		}
		b, ok := args[1].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "file.write_bytes: data must be bytes"}
		}
		if err := os.WriteFile(path, b.data, 0644); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return nil, nil
	}))
	env.set("file_copy", Builtin(func(args []Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, fmt.Errorf("file_copy expects 2 or 3 arguments")
		}
		src, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "filesystem.copy: src must be string"}
		}
		dst, ok := args[1].(string)
		if !ok {
			return nil, &raisedSignal{value: "filesystem.copy: dst must be string"}
		}
		overwrite := true
		preserveMode := true
		if len(args) == 3 && args[2] != nil {
			opts, ok := args[2].(Dict)
			if !ok {
				return nil, &raisedSignal{value: "filesystem.copy: options must be dictionary"}
			}
			if v, has := opts["overwrite"].(bool); has {
				overwrite = v
			}
			if v, has := opts["preserve_mode"].(bool); has {
				preserveMode = v
			}
		}
		if !overwrite {
			if _, err := os.Stat(dst); err == nil {
				return nil, &raisedSignal{value: "filesystem.copy: destination exists"}
			}
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, &raisedSignal{value: "filesystem.copy: " + err.Error()}
		}
		mode := os.FileMode(0644)
		if preserveMode {
			if info, err := os.Stat(src); err == nil {
				mode = info.Mode().Perm()
			}
		}
		if err := os.WriteFile(dst, data, mode); err != nil {
			return nil, &raisedSignal{value: "filesystem.copy: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("file_chmod", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_chmod expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "filesystem.chmod: path must be string"}
		}
		mode, ok := numberAsInt(args[1])
		if !ok {
			return nil, &raisedSignal{value: "filesystem.chmod: mode must be integer"}
		}
		if err := os.Chmod(path, os.FileMode(mode)); err != nil {
			return nil, &raisedSignal{value: "filesystem.chmod: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("file_temp", Builtin(func(args []Value) (Value, error) {
		prefix, suffix := "tya", ""
		if len(args) > 2 {
			return nil, fmt.Errorf("file_temp expects 0 to 2 arguments")
		}
		if len(args) >= 1 {
			var ok bool
			prefix, ok = args[0].(string)
			if !ok {
				return nil, &raisedSignal{value: "filesystem.temp: prefix must be string"}
			}
		}
		if len(args) == 2 {
			var ok bool
			suffix, ok = args[1].(string)
			if !ok {
				return nil, &raisedSignal{value: "filesystem.temp: suffix must be string"}
			}
		}
		f, err := os.CreateTemp("", prefix+"*"+suffix)
		if err != nil {
			return nil, &raisedSignal{value: "filesystem.temp: " + err.Error()}
		}
		path := f.Name()
		if err := f.Close(); err != nil {
			return nil, &raisedSignal{value: "filesystem.temp: " + err.Error()}
		}
		return path, nil
	}))
	env.set("stderr_write", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("stderr_write", args)
		if err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(os.Stderr, text)
		return nil, nil
	}))
	env.set("file_append", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_append expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.append: path must be a string"}
		}
		text, ok := args[1].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.append: text must be a string"}
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if _, err := f.WriteString(text); err != nil {
			_ = f.Close()
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := f.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return nil, nil
	}))
	env.set("compress_gzip", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.gzip", args)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		w := gzip.NewWriter(&out)
		if _, err := w.Write(data); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := w.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &Bytes{data: out.Bytes()}, nil
	}))
	env.set("compress_gunzip", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.gunzip", args)
		if err != nil {
			return nil, err
		}
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		out, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		return &Bytes{data: out}, nil
	}))
	env.set("compress_zlib", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.zlib", args)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		w := zlib.NewWriter(&out)
		if _, err := w.Write(data); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := w.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &Bytes{data: out.Bytes()}, nil
	}))
	env.set("compress_unzlib", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.unzlib", args)
		if err != nil {
			return nil, err
		}
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		out, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		return &Bytes{data: out}, nil
	}))
	env.set("io_stdin", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stdin expects 0 arguments")
		}
		return &IOStream{file: os.Stdin, readable: true, borrowed: true}, nil
	}))
	env.set("io_stdout", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stdout expects 0 arguments")
		}
		return &IOStream{file: os.Stdout, writable: true, borrowed: true}, nil
	}))
	env.set("io_stderr", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stderr expects 0 arguments")
		}
		return &IOStream{file: os.Stderr, writable: true, borrowed: true}, nil
	}))
	env.set("io_open", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_open expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "io.open: path must be a string"}
		}
		mode, ok := args[1].(string)
		if !ok {
			return nil, &raisedSignal{value: "io.open: mode must be a string"}
		}
		flag := 0
		readable := strings.Contains(mode, "r")
		writable := strings.Contains(mode, "w") || strings.Contains(mode, "a")
		switch {
		case strings.Contains(mode, "a"):
			flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		case strings.Contains(mode, "w"):
			flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		case strings.Contains(mode, "r"):
			flag = os.O_RDONLY
		default:
			return nil, &raisedSignal{value: "io.open: invalid mode"}
		}
		f, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &IOStream{file: f, binary: strings.Contains(mode, "b"), readable: readable, writable: writable}, nil
	}))
	streamArg := func(name string, args []Value) (*IOStream, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects stream argument", name)
		}
		s, ok := args[0].(*IOStream)
		if !ok || s == nil {
			return nil, &raisedSignal{value: name + ": argument must be a stream"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": stream is closed"}
		}
		return s, nil
	}
	env.set("io_stream_read", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_stream_read expects 2 arguments")
		}
		s, err := streamArg("io.read", args)
		if err != nil {
			return nil, err
		}
		if !s.readable {
			return nil, &raisedSignal{value: "io.read: stream is not readable"}
		}
		size, ok := numberAsInt(args[1])
		if !ok || size < 0 {
			return nil, &raisedSignal{value: "io.read: size must be non-negative"}
		}
		buf := make([]byte, int(size))
		n, rerr := s.file.Read(buf)
		if rerr != nil && rerr != io.EOF {
			return nil, &raisedSignal{value: rerr.Error()}
		}
		buf = buf[:n]
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("io_stream_read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("io_stream_read_line expects 1 argument")
		}
		s, err := streamArg("io.read_line", args)
		if err != nil {
			return nil, err
		}
		if !s.readable {
			return nil, &raisedSignal{value: "io.read_line: stream is not readable"}
		}
		var buf []byte
		tmp := make([]byte, 1)
		for {
			n, rerr := s.file.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[0])
				if tmp[0] == '\n' {
					break
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, &raisedSignal{value: rerr.Error()}
			}
		}
		if len(buf) == 0 {
			return nil, nil
		}
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("io_stream_eof", Builtin(func(args []Value) (Value, error) {
		_, err := streamArg("io.eof?", args)
		if err != nil {
			return nil, err
		}
		return false, nil
	}))
	env.set("io_stream_write", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_stream_write expects 2 arguments")
		}
		s, err := streamArg("io.write", args)
		if err != nil {
			return nil, err
		}
		if !s.writable {
			return nil, &raisedSignal{value: "io.write: stream is not writable"}
		}
		var data []byte
		if b, ok := args[1].(*Bytes); ok {
			data = b.data
		} else {
			data = []byte(stringify(args[1]))
		}
		n, werr := s.file.Write(data)
		if werr != nil {
			return nil, &raisedSignal{value: werr.Error()}
		}
		return int64(n), nil
	}))
	env.set("io_stream_flush", Builtin(func(args []Value) (Value, error) {
		s, err := streamArg("io.flush", args)
		if err != nil {
			return nil, err
		}
		return nil, s.file.Sync()
	}))
	env.set("io_stream_close", Builtin(func(args []Value) (Value, error) {
		s, err := streamArg("io.close", args)
		if err != nil {
			return nil, err
		}
		s.closed = true
		if s.borrowed {
			return nil, nil
		}
		return nil, s.file.Close()
	}))
	socketOptions := func(options Value) (bool, time.Duration) {
		opts, _ := options.(Dict)
		binary := false
		timeout := time.Duration(0)
		if opts != nil {
			if mode, ok := opts["mode"].(string); ok && mode == "binary" {
				binary = true
			}
			if seconds, ok := numberAsFloat(opts["timeout"]); ok && seconds > 0 {
				timeout = time.Duration(seconds * float64(time.Second))
			}
		}
		return binary, timeout
	}
	socketArg := func(name string, args []Value) (*TCPSocket, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects socket argument", name)
		}
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.conn == nil {
			return nil, &raisedSignal{value: name + ": argument must be a socket"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": socket is closed"}
		}
		return s, nil
	}
	serverArg := func(name string, args []Value) (*TCPSocket, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects server argument", name)
		}
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.listener == nil {
			return nil, &raisedSignal{value: name + ": argument must be a socket server"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": socket is closed"}
		}
		return s, nil
	}
	socketAddress := func(addr net.Addr) Dict {
		host := ""
		port := int64(0)
		if tcp, ok := addr.(*net.TCPAddr); ok {
			host = tcp.IP.String()
			port = int64(tcp.Port)
		} else if addr != nil {
			host, _, _ = net.SplitHostPort(addr.String())
		}
		return Dict{"host": host, "port": port}
	}
	env.set("socket_connect", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("socket_connect expects 3 arguments")
		}
		host, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "socket.connect: host must be a string"}
		}
		port, ok := numberAsInt(args[1])
		if !ok || port < 0 || port > 65535 {
			return nil, &raisedSignal{value: "socket.connect: invalid port"}
		}
		binary, timeout := socketOptions(args[2])
		address := net.JoinHostPort(host, strconv.FormatInt(port, 10))
		var conn net.Conn
		var err error
		if timeout > 0 {
			conn, err = net.DialTimeout("tcp", address, timeout)
		} else {
			conn, err = net.Dial("tcp", address)
		}
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{conn: conn, binary: binary, timeout: timeout}, nil
	}))
	env.set("socket_server_listen", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("socket_server_listen expects 3 arguments")
		}
		host, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "socket.listen: host must be a string"}
		}
		port, ok := numberAsInt(args[1])
		if !ok || port < 0 || port > 65535 {
			return nil, &raisedSignal{value: "socket.listen: invalid port"}
		}
		binary, timeout := socketOptions(args[2])
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.FormatInt(port, 10)))
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{listener: listener, binary: binary, timeout: timeout}, nil
	}))
	env.set("socket_server_accept", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("socket_server_accept expects 1 argument")
		}
		s, err := serverArg("socket.accept", args)
		if err != nil {
			return nil, err
		}
		if s.timeout > 0 {
			if tcp, ok := s.listener.(*net.TCPListener); ok {
				_ = tcp.SetDeadline(time.Now().Add(s.timeout))
			}
		}
		conn, err := s.listener.Accept()
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{conn: conn, binary: s.binary, timeout: s.timeout}, nil
	}))
	env.set("socket_read", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("socket_read expects 2 arguments")
		}
		s, err := socketArg("socket.read", args)
		if err != nil {
			return nil, err
		}
		size, ok := numberAsInt(args[1])
		if !ok || size < 0 {
			return nil, &raisedSignal{value: "socket.read: size must be non-negative"}
		}
		if s.timeout > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		}
		buf := make([]byte, int(size))
		n, err := s.conn.Read(buf)
		if err != nil && err != io.EOF {
			return nil, &raisedSignal{value: err.Error()}
		}
		buf = buf[:n]
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("socket_read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("socket_read_line expects 1 argument")
		}
		s, err := socketArg("socket.read_line", args)
		if err != nil {
			return nil, err
		}
		if s.timeout > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		}
		var buf []byte
		tmp := make([]byte, 1)
		for {
			n, rerr := s.conn.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[0])
				if tmp[0] == '\n' {
					break
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, &raisedSignal{value: rerr.Error()}
			}
		}
		if len(buf) == 0 {
			return nil, nil
		}
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("socket_write", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("socket_write expects 2 arguments")
		}
		s, err := socketArg("socket.write", args)
		if err != nil {
			return nil, err
		}
		var data []byte
		if b, ok := args[1].(*Bytes); ok {
			data = b.data
		} else {
			data = []byte(stringify(args[1]))
		}
		if s.timeout > 0 {
			_ = s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
		}
		n, err := s.conn.Write(data)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return int64(n), nil
	}))
	env.set("socket_close", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.conn == nil || s.closed {
			return nil, nil
		}
		s.closed = true
		return nil, s.conn.Close()
	}))
	env.set("socket_closed", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		return !ok || s == nil || s.closed, nil
	}))
	env.set("socket_local_address", Builtin(func(args []Value) (Value, error) {
		s, err := socketArg("socket.local_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.conn.LocalAddr()), nil
	}))
	env.set("socket_remote_address", Builtin(func(args []Value) (Value, error) {
		s, err := socketArg("socket.remote_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.conn.RemoteAddr()), nil
	}))
	env.set("socket_server_close", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.listener == nil || s.closed {
			return nil, nil
		}
		s.closed = true
		return nil, s.listener.Close()
	}))
	env.set("socket_server_local_address", Builtin(func(args []Value) (Value, error) {
		s, err := serverArg("socket.server.local_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.listener.Addr()), nil
	}))
}

func numberAsFloat(v Value) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func valueBytes(name string, args []Value) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	switch v := args[0].(type) {
	case string:
		return []byte(v), nil
	case *Bytes:
		return v.data, nil
	default:
		return nil, &raisedSignal{value: name + ": value must be a string or bytes"}
	}
}

func numberAsInt(v Value) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x != mathpkg.Trunc(x) {
			return 0, false
		}
		return int64(x), true
	}
	return 0, false
}
