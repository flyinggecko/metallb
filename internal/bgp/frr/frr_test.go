// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"go.universe.tf/metallb/internal/bgp"
)

const testData = "testdata/"

var classCMask = net.IPv4Mask(0xff, 0xff, 0xff, 0)

var update = flag.Bool("update", false, "update .golden files")

func testOsHostname() (string, error) {
	return "dummyhostname", nil
}

func testCompareFiles(t *testing.T, configFile, goldenFile string) {
	cmd := exec.Command("diff", configFile, goldenFile)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("command %s returned error: %s\n%s", cmd.String(), err, output)
	}
}

func testUpdateGoldenFile(t *testing.T, configFile, goldenFile string) {
	t.Log("update golden file")
	cmd := exec.Command("cp", "-a", configFile, goldenFile)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("command %s returned %s and error: %s", cmd.String(), output, err)
	}
}

func testGenerateFileNames(t *testing.T) (string, string) {
	return filepath.Join(testData, filepath.FromSlash(t.Name())), filepath.Join(testData, filepath.FromSlash(t.Name())+".golden")
}

func testSetup(t *testing.T) {
	configFile, _ := testGenerateFileNames(t)

	_, err := os.Create(configFile)
	if err != nil {
		t.Fatalf("cannot create config file %s: %s", configFile, err)
	}

	osHostname = testOsHostname

	os.Setenv("FRR_CONFIG_FILE", configFile)
	// override reloadConfig so it doesn't try to reload it.
	reloadConfig = func() error { return nil }
}

func testCheckConfigFile(t *testing.T) {
	configFile, goldenFile := testGenerateFileNames(t)

	if *update {
		testUpdateGoldenFile(t, configFile, goldenFile)
	}
	testCompareFiles(t, configFile, goldenFile)
	if !strings.Contains(configFile, "Invalid") {
		err := testFileIsValid(configFile)
		if err != nil {
			t.Fatalf("Failed to verify the file %s", err)
		}
	}
}

func TestSingleSession(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	testCheckConfigFile(t)
}

func TestSingleSessionClose(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}

	session.Close()
	testCheckConfigFile(t)
}
func TestTwoSessions(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session1, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session1.Close()
	session2, err := sessionManager.NewSession(l, "10.4.4.254:179", net.ParseIP("10.3.3.254"), 300, net.ParseIP("10.3.3.254"), 400, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session2.Close()

	testCheckConfigFile(t)
}

func TestTwoSessionsDuplicate(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session1, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session1.Close()
	session2, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session2.Close()

	testCheckConfigFile(t)
}

func TestTwoSessionsDuplicateRouter(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session1, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session1.Close()
	session2, err := sessionManager.NewSession(l, "10.4.4.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 400, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session2.Close()

	testCheckConfigFile(t)
}

func TestSingleAdvertisement(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv := &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementNoRouterID(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, nil, 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv := &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementInvalidPrefix(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{}

	adv := &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err == nil {
		t.Fatalf("Set should return error")
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementInvalidNoPort(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err == nil {
		session.Close()
		t.Fatalf("Should not be able to create session")
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementInvalidNextHop(t *testing.T) {
	t.Skip("TODO: bgp.Validate() incorrectly(?) returns err == nil")
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv := &bgp.Advertisement{
		Prefix: prefix,
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementStop(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv := &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	err = session.Set()
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}

func TestSingleAdvertisementChange(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv := &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	prefix = &net.IPNet{
		IP:   net.ParseIP("172.16.1.11"),
		Mask: classCMask,
	}

	adv = &bgp.Advertisement{
		Prefix:  prefix,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}

func TestTwoAdvertisements(t *testing.T) {
	testSetup(t)

	l := log.NewNopLogger()
	sessionManager := NewSessionManager()
	session, err := sessionManager.NewSession(l, "10.2.2.254:179", net.ParseIP("10.1.1.254"), 100, net.ParseIP("10.1.1.254"), 200, time.Second, time.Second, "password", "hostname", "")
	if err != nil {
		t.Fatalf("Could not create session: %s", err)
	}
	defer session.Close()

	prefix1 := &net.IPNet{
		IP:   net.ParseIP("172.16.1.10"),
		Mask: classCMask,
	}

	adv1 := &bgp.Advertisement{
		Prefix:  prefix1,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	prefix2 := &net.IPNet{
		IP:   net.ParseIP("172.16.1.11"),
		Mask: classCMask,
	}

	adv2 := &bgp.Advertisement{
		Prefix:  prefix2,
		NextHop: net.ParseIP("10.1.1.1"),
	}

	err = session.Set(adv1, adv2)
	if err != nil {
		t.Fatalf("Could not advertise prefix: %s", err)
	}

	testCheckConfigFile(t)
}
