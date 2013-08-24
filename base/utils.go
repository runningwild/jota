package base

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/runningwild/glop/gui"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"sync"
	"time"
)

var datadir string
var logger *log.Logger
var log_reader io.Reader
var log_out *os.File

var log_console *bytes.Buffer
var logTailer Tailer

func GetLogTailer() Tailer {
	if logTailer == nil {
		panic("Can't get the log Tailer before the logging has been set up.")
	}
	return logTailer
}

func SetDatadir(_datadir string) {
	datadir = _datadir
	setupLogger()
}
func GetDataDir() string {
	return datadir
}

func setupLogger() {
	// If an error happens when making this directory it might already exist,
	// all that really matters is making the log file in the directory.
	os.Mkdir(filepath.Join(datadir, "logs"), 0777)
	logger = nil
	var err error
	name := time.Now().Format("2006-01-02-15-04-05") + ".log"
	log_out, err = os.Create(filepath.Join(datadir, "logs", name))
	if err != nil {
		fmt.Printf("Unable to open log file: %v\nLogging to stdout...\n", err.Error())
		log_out = os.Stdout
	}
	tee := bytes.NewBuffer(nil)
	logWriter := io.MultiWriter(tee, log_out)
	logTailer = newTail(tee, 100)
	logger = log.New(logWriter, "> ", log.Ltime|log.Lshortfile)
}

// TODO: This probably isn't the best way to do things - different go-routines
// can call these and screw up prefixes for each other.
func Log() *log.Logger {
	logger.SetPrefix("LOG  > ")
	return logger
}

func Warn() *log.Logger {
	logger.SetPrefix("WARN > ")
	return logger
}

func Error() *log.Logger {
	logger.SetPrefix("ERROR> ")
	return logger
}

func CloseLog() {
	log_out.WriteString("END OF LOG\n\n\n\n")
	log_out.Close()
}

var font_dict map[string]*gui.Dictionary
var dictionary_mutex sync.Mutex

func LoadAllDictionaries() {
	filenames, err := filepath.Glob(filepath.Join(GetDataDir(), "fonts", "*.gob"))
	if err != nil {
		Log().Fatalf("Unable to open font dirs: %v", err)
	}
	for _, filename := range filenames {
		font := filepath.Base(filename)
		font = font[0 : len(font)-4]
		GetDictionary(font)
	}
}

func GetDictionary(font string) *gui.Dictionary {
	dictionary_mutex.Lock()
	defer dictionary_mutex.Unlock()
	if font_dict == nil {
		font_dict = make(map[string]*gui.Dictionary)
	}
	if _, ok := font_dict[font]; !ok {
		path := filepath.Join(datadir, "fonts", font+".gob")
		f, err := os.Open(path)
		if err != nil {
			Error().Printf("Unable to open file '%s': %v\n", path, err)
			return nil
		}
		defer f.Close()
		dict, err := gui.LoadDictionary(f)
		if err != nil {
			Error().Printf("Unable to load dictionary from '%s': %v\n", path, err)
			return nil
		}
		font_dict[font] = dict
	}
	return font_dict[font]
}

// A Path is a string that is intended to store a path.  When it is encoded
// with gob or json it will convert itself to a relative path relative to
// datadir.  When it is decoded from gob or json it will convert itself to an
// absolute path based on datadir.
type Path string

func (p Path) String() string {
	return string(p)
}
func (p Path) GobEncode() ([]byte, error) {
	return []byte(TryRelative(datadir, string(p))), nil
}
func (p *Path) GobDecode(data []byte) error {
	*p = Path(filepath.Join(datadir, string(data)))
	return nil
}
func (p Path) MarshalJSON() ([]byte, error) {
	val := filepath.ToSlash(TryRelative(datadir, string(p)))
	return []byte("\"" + val + "\""), nil
}
func (p *Path) UnmarshalJSON(data []byte) error {
	rel := filepath.FromSlash(string(data[1 : len(data)-1]))
	*p = Path(filepath.Join(datadir, rel))
	return nil
}

// Opens the file named by path, reads it all, decodes it as json into target,
// then closes the file.  Returns the first error found while doing this or nil.
func LoadJson(path string, target interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	return err
}

func SaveJson(path string, source interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func ToGobToBase64(src interface{}) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(src)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func FromBase64FromGob(dst interface{}, str string) error {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(dst)
}

// Opens the file named by path, reads it all, decodes it as gob into target,
// then closes the file.  Returns the first error found while doing this or nil.
func LoadGob(path string, target interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	err = dec.Decode(target)
	return err
}

func SaveGob(path string, source interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(source)
	return err
}

// Returns a path rel such that filepath.Join(a, rel) and b refer to the same
// file.  a and b must both be relative paths or both be absolute paths.  If
// they are not then b will be returned in either case.
func TryRelative(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err == nil {
		return rel
	}
	return target
}

func GetStoreVal(key string) string {
	var store map[string]string
	LoadJson(filepath.Join(datadir, "store"), &store)
	if store == nil {
		store = make(map[string]string)
	}
	val := store[key]
	return val
}

func SetStoreVal(key, val string) {
	var store map[string]string
	path := filepath.Join(datadir, "store")
	LoadJson(path, &store)
	if store == nil {
		store = make(map[string]string)
	}
	store[key] = val
	SaveJson(path, store)
}

type sortAnythings struct {
	values []reflect.Value
	less   reflect.Value
}

func (s sortAnythings) Len() int { return len(s.values) }
func (s sortAnythings) Less(i, j int) bool {
	return s.less.Call([]reflect.Value{s.values[i], s.values[j]})[0].Bool()
}
func (s sortAnythings) Swap(i, j int) { s.values[i], s.values[j] = s.values[j], s.values[i] }

// This is a slowish way to iterate through a map in a deterministic order.
// If this is ever too slow then either a non-reflecty version should be made.
func DoOrdered(mapIn interface{}, less interface{}, do interface{}) {
	mapInValue := reflect.ValueOf(mapIn)
	if mapInValue.Kind() != reflect.Map {
		panic(fmt.Sprintf("Parameter 'mapIn' to iterateOrdered must be a map, not a %v", mapInValue.Kind()))
	}

	lessValue := reflect.ValueOf(less)
	if lessValue.Kind() != reflect.Func || lessValue.Type().NumIn() != 2 || lessValue.Type().NumOut() != 1 {
		panic("Parameter 'less' to iterateOrdered must be a function with two inputs and out output.")
	}
	if lessValue.Type().Out(0).Kind() != reflect.Bool {
		panic("Parameter 'less' to iterateOrdered must return a bool.")
	}
	if lessValue.Type().In(0) != lessValue.Type().In(1) || lessValue.Type().In(0) != mapInValue.Type().Key() {
		panic("Parameter 'less' to iterateOrdered must take two parameters with the same type as the values of the 'mapIn' parameter.")
	}

	doValue := reflect.ValueOf(do)
	if doValue.Kind() != reflect.Func || doValue.Type().NumIn() != 2 {
		panic(fmt.Sprintf("Parameter 'do' to iterateOrdered must be a function of two parameters, not a %v", doValue.Kind()))
	}
	if doValue.Type().In(0) != mapInValue.Type().Key() || doValue.Type().In(1) != mapInValue.Type().Elem() {
		panic("The first and second parameters to do must match the types of the key and values of mapIn, respectively.")
	}

	keys := mapInValue.MapKeys()
	if len(keys) == 0 {
		return
	}

	sort.Sort(sortAnythings{keys, lessValue})
	for _, key := range keys {
		value := mapInValue.MapIndex(key)
		doValue.Call([]reflect.Value{key, value})
	}
}

func StackCatcher() {
	if r := recover(); r != nil {
		Error().Printf("Panic: %v", r)
		Error().Fatalf("Stack:\n%s", debug.Stack())
	}
}

func GoWithStackCatcher(f_ interface{}, inputs_ ...interface{}) {
	f := reflect.ValueOf(f_)
	if f.Kind() != reflect.Func {
		panic(fmt.Sprintf("First parameter to GoWithStackCatcher must be a Func, not %v", f.Kind()))
	}
	var inputs []reflect.Value
	for i := range inputs_ {
		inputs = append(inputs, reflect.ValueOf(inputs_[i]))
	}
	go func() {
		defer StackCatcher()
		f.Call(inputs)
	}()
}
