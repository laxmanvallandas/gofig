package gofig

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	//jww "github.com/spf13/jwalterweatherman"
)

var (
	tmpPrefixDirs []string
)

func TestMain(m *testing.M) {
	if debug, _ := strconv.ParseBool(os.Getenv("GOFIG_DEBUG")); debug {
		log.SetLevel(log.DebugLevel)
		//jww.SetStdoutThreshold(jww.LevelTrace)
	}
	Register(testReg1())
	Register(testReg2())

	exitCode := m.Run()
	for _, d := range tmpPrefixDirs {
		os.RemoveAll(d)
	}
	os.Exit(exitCode)
}

func newConfigDirs(testName string, t *testing.T) (string, string) {
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("gofig-test-%s", testName))
	if err != nil {
		t.Fatal(err)
	}

	etcDirPath := fmt.Sprintf("%s/etc/gofig", tmpDir)
	usrDirPath := fmt.Sprintf("%s/home/gofig", tmpDir)
	SetGlobalConfigPath(etcDirPath)
	SetUserConfigPath(usrDirPath)

	os.MkdirAll(etcDirPath, 0755)
	os.MkdirAll(usrDirPath, 0755)

	etcFilePath := fmt.Sprintf("%s/config.yml", etcDirPath)
	usrFilePath := fmt.Sprintf("%s/config.yml", usrDirPath)

	tmpPrefixDirs = append(tmpPrefixDirs, tmpDir)
	return etcFilePath, usrFilePath
}

func assertConfigEqualToJSON(
	c1 Config, j2 string, t *testing.T) (Config, Config, bool) {
	var err error
	var j1 string
	if j1, err = c1.ToJSON(); err != nil {
		t.Error(err)
		return nil, nil, false
	}
	return assertJSONEqual(j1, j2, t)
}

func assertConfigEqualToJSONCompact(
	c1 Config, j2 string, t *testing.T) (Config, Config, bool) {
	var err error
	var j1 string
	if j1, err = c1.ToJSONCompact(); err != nil {
		t.Error(err)
		return nil, nil, false
	}
	return assertJSONEqual(j1, j2, t)
}

func assertJSONEqual(
	j1 string, j2 string, t *testing.T) (Config, Config, bool) {

	t.Logf("j1 - %s", j1)
	t.Log("")
	t.Logf("j2 - %s", j2)
	t.Log("")

	c1, error := FromJSON(j1)
	if error != nil {
		t.Errorf("error reading JSON %s %v", j1, error)
		return nil, nil, false
	}

	c2, error := FromJSON(j2)
	if error != nil {
		t.Errorf("error reading JSON %s %v", j2, error)
		return nil, nil, false
	}

	eq := assertConfigsEqual(c1, c2, t)

	return c1, c2, eq
}

func assertConfigsEqual(c1 Config, c2 Config, t *testing.T) bool {

	printConfig("c1", c1, t)
	t.Log("")
	printConfig("c2", c2, t)
	t.Log("")

	c1Keys := c1.AllKeys()
	c2Keys := c2.AllKeys()

	for _, k := range c1Keys {
		c1v := c1.Get(k)
		c2v := c2.Get(k)
		if !reflect.DeepEqual(c1v, c2v) {
			t.Logf("%s != in both configs; c1v=%v, c2v=%v", k, c1v, c2v)
			return false
		}
	}

	for _, k := range c2Keys {
		c1v := c1.Get(k)
		c2v := c2.Get(k)
		if !reflect.DeepEqual(c1v, c2v) {
			t.Logf("%s != in both configs; c1v=%v, c2v=%v", k, c1v, c2v)
			return false
		}
	}

	return true
}

func TestAssertConfigDefaults(t *testing.T) {
	newConfigDirs("TestAssertConfigDefaults", t)
	wipeEnv()
	c := New()

	osDrivers := c.GetStringSlice("rexray.osDrivers")
	volDrivers := c.GetStringSlice("rexray.volumeDrivers")

	assertString(t, c, "rexray.host", "tcp://:7979")
	assertString(t, c, "rexray.logLevel", "warn")

	if len(osDrivers) != 1 || osDrivers[0] != "linux" {
		t.Fatalf("osDrivers != []string{\"linux\"}, == %v", osDrivers)
	}

	if len(volDrivers) != 1 || volDrivers[0] != "docker" {
		t.Fatalf("volumeDrivers != []string{\"docker\"}, == %v", volDrivers)
	}
}

func TestAssertTestRegistration(t *testing.T) {
	newConfigDirs("TestAssertTestRegistration", t)
	wipeEnv()
	Register(testReg3())
	c := New()
	printConfig("", c, t)

	userName := c.GetString("mockProvider.username")
	password := c.GetString("mockProvider.password")
	useCerts := c.GetBool("mockProvider.useCerts")
	minVolSize := c.GetInt("mockProvider.Docker.minVolSize")

	if userName != "admin" {
		t.Errorf("mockProvider.userName != admin, == %s", userName)
	}

	if password != "" {
		t.Errorf("mockProvider.password != '', == %s", password)
	}

	if !useCerts {
		t.Errorf("mockProvider.useCerts != true, == %v", useCerts)
	}

	if minVolSize != 16 {
		t.Errorf("minVolSize != 16, == %d", minVolSize)
	}
}

func TestBaselineJSON(t *testing.T) {
	newConfigDirs("TestBaselineJSON", t)
	wipeEnv()
	Register(testReg3())
	c := New()
	assertConfigEqualToJSON(c, jsonConfigBaseline, t)
}

func TestToJSON(t *testing.T) {
	newConfigDirs("TestToJSON", t)
	wipeEnv()
	Register(testReg3())
	c := New()

	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}

	c1, c2, eq := assertConfigEqualToJSON(c, jsonConfigWithYamlConfig1, t)
	if !eq {
		t.Fatal("configs not equal pre minVolSize change")
	}
	t.Log("configs equal pre minVolSize change")

	mvs := c1.GetInt("mockprovider.docker.minvolsize")
	if mvs != 32 {
		t.Fatal("mvs != 32")
	}

	c1.Set("mockprovider.docker.minvolsize", 128)
	mvs = c1.GetInt("mockprovider.docker.minvolsize")
	if mvs != 128 {
		t.Fatal("mvs != 128")
	}

	if eq := assertConfigsEqual(c1, c2, t); eq {
		t.Fatal("configs equal post minVolSize change")
	}
}

func TestToJSONCompact(t *testing.T) {
	newConfigDirs("TestToJSONCompact", t)
	wipeEnv()
	Register(testReg3())
	c := New()

	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}

	c1, c2, eq := assertConfigEqualToJSONCompact(
		c, jsonConfigWithYamlConfig1, t)
	if !eq {
		t.Fatal("configs not equal pre minVolSize change")
	}
	t.Log("configs equal pre minVolSize change")

	mvs := c1.GetInt("mockprovider.docker.minvolsize")
	if mvs != 32 {
		t.Fatal("mvs != 32")
	}

	c1.Set("mockprovider.docker.minvolsize", 128)
	mvs = c1.GetInt("mockprovider.docker.minvolsize")
	if mvs != 128 {
		t.Fatal("mvs != 128")
	}

	if eq := assertConfigsEqual(c1, c2, t); eq {
		t.Fatal("configs equal post minVolSize change")
	}
}

func TestFromJSON(t *testing.T) {
	newConfigDirs("TestFromJSON", t)
	wipeEnv()
	Register(testReg3())
	c := New()
	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}
	assertConfigEqualToJSON(c, jsonConfigWithYamlConfig1, t)
}

func TestFromJSONWithErrors(t *testing.T) {
	_, err := FromJSON("///*")
	if err == nil {
		t.Fatal("expected unmarshalling error")
	}
}

func TestEnvVars(t *testing.T) {
	newConfigDirs("TestEnvVars", t)
	wipeEnv()
	Register(testReg3())
	c := New()

	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}

	fev := c.EnvVars()

	for _, v := range fev {
		t.Log(v)
	}

	assertEnvVar("REXRAY_HOST=tcp://:7979", fev, t)
	assertEnvVar("REXRAY_LOGLEVEL=error", fev, t)
	assertEnvVar("REXRAY_STORAGEDRIVERS=ec2 xtremio", fev, t)
	assertEnvVar("REXRAY_OSDRIVERS=linux", fev, t)
	assertEnvVar("REXRAY_VOLUMEDRIVERS=docker", fev, t)
	assertEnvVar("MOCKPROVIDER_USERNAME=admin", fev, t)
	assertEnvVar("MOCKPROVIDER_USECERTS=true", fev, t)
	assertEnvVar("MOCKPROVIDER_DOCKER_MINVOLSIZE=32", fev, t)
}

func assertEnvVar(s string, evs []string, t *testing.T) {
	if !stringInSlice(s, evs) {
		t.Fatal(s)
	}
}

func TestCopy(t *testing.T) {
	etcCfgFilePath, _ := newConfigDirs("TestCopy", t)
	wipeEnv()
	Register(testReg3())

	t.Logf("etcCfgFilePath=%s", etcCfgFilePath)
	writeStringToFile(string(yamlConfig1), etcCfgFilePath)

	c1 := New()

	assertString(t, c1, "rexray.logLevel", "error")
	assertStorageDrivers(t, c1)
	assertOsDrivers1(t, c1)

	c2, _ := c1.Copy()

	assertString(t, c2, "rexray.logLevel", "error")
	assertStorageDrivers(t, c2)
	assertOsDrivers1(t, c2)

	assertConfigsEqual(c1, c2, t)
}

func TestNewWithUserConfigFile(t *testing.T) {
	_, usrCfgFilePath := newConfigDirs("TestNewWithUserConfigFile", t)
	writeStringToFile(string(yamlConfig1), usrCfgFilePath)

	c := New()

	assertString(t, c, "rexray.logLevel", "error")
	assertStorageDrivers(t, c)
	assertOsDrivers1(t, c)

	if err := c.ReadConfig(bytes.NewReader(yamlConfig2)); err != nil {
		t.Fatal(err)
	}

	assertString(t, c, "rexray.logLevel", "debug")
	assertStorageDrivers(t, c)
	assertOsDrivers2(t, c)
}

func TestNewWithUserConfigFileWithErrors(t *testing.T) {
	_, usrCfgFilePath := newConfigDirs("TestNewWithUserConfigFileWithErrors", t)
	writeStringToFile(string(yamlConfig1), usrCfgFilePath)

	os.Chmod(usrCfgFilePath, 0000)
	New()
}

func TestNewWithGlobalConfigFile(t *testing.T) {
	etcCfgFilePath, _ := newConfigDirs("TestNewWithGlobalConfigFile", t)

	t.Logf("etcCfgFilePath=%s", etcCfgFilePath)
	writeStringToFile(string(yamlConfig1), etcCfgFilePath)

	c := New()

	assertString(t, c, "rexray.logLevel", "error")
	assertStorageDrivers(t, c)
	assertOsDrivers1(t, c)

	if err := c.ReadConfig(bytes.NewReader(yamlConfig2)); err != nil {
		t.Fatal(err)
	}

	assertString(t, c, "rexray.logLevel", "debug")
	assertStorageDrivers(t, c)
	assertOsDrivers2(t, c)
}

func TestNewWithGlobalConfigFileWithErrors(t *testing.T) {
	etcCfgFilePath, _ := newConfigDirs(
		"TestNewWithGlobalConfigFileWithErrors", t)

	t.Logf("etcCfgFilePath=%s", etcCfgFilePath)
	writeStringToFile(string(yamlConfig1), etcCfgFilePath)

	os.Chmod(etcCfgFilePath, 0000)
	New()
}

func TestReadConfigFile(t *testing.T) {
	var err error
	var tmp *os.File
	if tmp, err = ioutil.TempFile("", "TestReadConfigFile"); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(yamlConfig1); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	os.Chmod(tmp.Name(), 0000)

	c := New()
	if err := c.ReadConfigFile(tmp.Name()); err == nil {
		t.Fatal("expected error reading config file")
	}
}

func TestReadConfig(t *testing.T) {
	c := NewConfig(false, false, "config", "yml")
	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}

	assertString(t, c, "rexray.logLevel", "error")
	assertStorageDrivers(t, c)
	assertOsDrivers1(t, c)

	if err := c.ReadConfig(bytes.NewReader(yamlConfig2)); err != nil {
		t.Fatal(err)
	}

	assertString(t, c, "rexray.logLevel", "debug")
	assertStorageDrivers(t, c)
	assertOsDrivers2(t, c)
}

func TestReadNilConfig(t *testing.T) {
	if err := New().ReadConfig(nil); err == nil {
		t.Fatal("expected nil config error")
	}
}

func TestScope(t *testing.T) {
	wipeEnv()
	Register(testReg3())
	c := New()
	assert.True(t, c.IsSet("rexray.loglevel"))
	assert.Equal(t, "warn", c.GetString("rexray.loglevel"))

	if err := c.ReadConfig(bytes.NewReader(yamlConfig1)); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "error", c.GetString("rexray.loglevel"))

	c.Set("loglevel", "verbose")

	assert.True(t, c.IsSet("loglevel"))
	assert.Equal(t, "verbose", c.GetString("loglevel"))

	sc := c.Scope("rexray")

	assert.True(t, sc.IsSet("loglevel"))
	assert.Equal(t, "error", sc.GetString("loglevel"))
}

func wipeEnv() {
	evs := os.Environ()
	for _, v := range evs {
		k := strings.Split(v, "=")[0]
		os.Setenv(k, "")
	}
}

func printKeys(title string, c Config, t *testing.T) {
	for _, k := range c.AllKeys() {
		if title == "" {
			t.Logf(k)
		} else {
			t.Logf("%s - %s", title, k)
		}
	}
}

func printViperKeys(title string, c Config, t *testing.T) {
	tc := c.(*config)
	for _, k := range tc.v.AllKeys() {
		if title == "" {
			t.Logf(k)
		} else {
			t.Logf("%s - %s", title, k)
		}
	}
}

func printConfig(title string, c Config, t *testing.T) {
	for _, k := range c.AllKeys() {
		if title == "" {
			t.Logf("%s=%v", k, c.Get(k))
		} else {
			t.Logf("%s - %s=%v", title, k, c.Get(k))
		}
	}
}

func assertString(t *testing.T, c Config, key, expected string) {
	v := c.GetString(key)
	if v != expected {
		t.Fatalf("%s != %s; == %v", key, expected, v)
	}
}

func assertStorageDrivers(t *testing.T, c Config) {
	sd := c.GetStringSlice("rexray.storageDrivers")
	if sd == nil {
		t.Fatalf("storageDrivers == nil")
	}

	if len(sd) != 2 {
		t.Fatalf("len(storageDrivers) != 2; == %d", len(sd))
	}

	if sd[0] != "ec2" {
		t.Fatalf("sd[0] != ec2; == %v", sd[0])
	}

	if sd[1] != "xtremio" {
		t.Fatalf("sd[1] != xtremio; == %v", sd[1])
	}
}

func assertOsDrivers1(t *testing.T, c Config) {
	od := c.GetStringSlice("rexray.osDrivers")
	if od == nil {
		t.Fatalf("osDrivers == nil")
	}
	if len(od) != 1 {
		t.Fatalf("len(osDrivers) != 1; == %d", len(od))
	}
	if od[0] != "linux" {
		t.Fatalf("od[0] != linux; == %v", od[0])
	}
}

func assertOsDrivers2(t *testing.T, c Config) {
	od := c.GetStringSlice("rexray.osDrivers")
	if od == nil {
		t.Fatalf("osDrivers == nil")
	}
	if len(od) != 2 {
		t.Fatalf("len(osDrivers) != 2; == %d", len(od))
	}
	if od[0] != "darwin" {
		t.Fatalf("od[0] != darwin; == %v", od[0])
	}
	if od[1] != "linux" {
		t.Fatalf("od[1] != linux; == %v", od[1])
	}
}

var yamlConfig1 = []byte(`
rexray:
    logLevel: error
    storageDrivers:
    - ec2
    - xtremio
    osDrivers:
    - linux
mockProvider:
  userName: admin
  useCerts: true
  docker:
    MinVolSize: 32
`)

var yamlConfig2 = []byte(`
rexray:
    logLevel: debug
    osDrivers:
    - darwin
    - linux
`)

var jsonConfigBaseline = `{
    "mockprovider": {
        "docker": {
            "MinVolSize": 16
        },
        "useCerts": true,
        "userName": "admin"
    },
    "mockprovider.docker.maxvolsize": 256,
    "mockprovider.insecure": true,
    "mockprovider.password": "",
    "rexray": {
        "osDrivers": [
            "linux"
        ],
        "storageDrivers": [
            "libstorage"
        ],
        "volumeDrivers": [
            "docker"
        ]
    },
    "rexray.host": "tcp://:7979",
    "rexray.loglevel": "warn"
}
`

var jsonConfigWithYamlConfig1 = `{
    "mockprovider": {
        "docker": {
            "MinVolSize": 32
        },
        "useCerts": true,
        "userName": "admin"
    },
    "mockprovider.docker.maxvolsize": 256,
    "mockprovider.insecure": true,
    "mockprovider.password": "",
    "rexray": {
        "osDrivers": [
            "linux"
        ],
        "storageDrivers": [
            "ec2",
            "xtremio"
        ],
        "volumeDrivers": [
            "docker"
        ]
    },
    "rexray.host": "tcp://:7979",
    "rexray.loglevel": "error"
}
`

func writeStringToFile(text, path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		return err
	}

	f.WriteString(text)
	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if strings.ToLower(b) == strings.ToLower(a) {
			return true
		}
	}
	return false
}

func testReg1() *Registration {
	r := NewRegistration("Global")
	r.Yaml(`
rexray:
    host: tcp://:7979
    logLevel: warn
`)
	r.Key(String, "h", "tcp://:7979",
		"The REX-Ray host", "rexray.host")
	r.Key(String, "l", "warn",
		"The log level (error, warn, info, debug)", "rexray.logLevel")
	return r
}

func testReg2() *Registration {
	r := NewRegistration("Driver")
	r.Yaml(`
rexray:
    osDrivers:
    - linux
    storageDrivers:
    - libstorage
    volumeDrivers:
    - docker
`)
	r.Key(String, "", "linux",
		"The OS drivers to consider", "rexray.osDrivers")
	r.Key(String, "", "libstorage",
		"The storage drivers to consider", "rexray.storageDrivers")
	r.Key(String, "", "docker",
		"The volume drivers to consider", "rexray.volumeDrivers")
	return r
}

func testReg3() *Registration {
	r := NewRegistration("Mock Provider")
	r.Yaml(`mockProvider:
    userName: admin
    useCerts: true
    docker:
        MinVolSize: 16
`)
	r.Key(String, "", "admin", "", "mockProvider.userName")
	r.Key(String, "", "", "", "mockProvider.password")
	r.Key(Bool, "", false, "", "mockProvider.useCerts")
	r.Key(Int, "", 16, "", "mockProvider.docker.minVolSize")
	r.Key(Bool, "i", true, "", "mockProvider.insecure")
	r.Key(Int, "m", 256, "", "mockProvider.docker.maxVolSize")
	return r
}

func testReg3a() *Registration {
	r := NewRegistration("Test Reg 3")
	r.Yaml(`testReg3:
    userName: admin
    useCerts: true
    keyFiles:
        pubKey: MyPubKey
		PrvKey: MyPrvKey
`)
	r.Key(String, "", "admin", "", "testReg3.userName")
	r.Key(String, "", "", "", "testReg3.password")
	r.Key(Bool, "", false, "", "testReg3.useCerts")
	r.Key(String, "", "", "", "testReg3.keyFiles.pubKey")
	r.Key(String, "", "", "", "testReg3.keyFiles.prvKey")
	return r
}