package test

import (
	"net"
	"os"
	"testing"

	"github.com/duke-git/lancet/netutil"
	"github.com/duke-git/lancet/v2/fileutil"
)

func TestCurrentPath(t *testing.T) {
	absPath := fileutil.CurrentPath()
	t.Log(absPath)
}

func TestFileMode(t *testing.T) {
	isCreate := fileutil.CreateFile("./test.txt")
	if !isCreate {
		t.Log("err: ", isCreate)
	}

	mode, err := fileutil.FileMode("./test.txt")
	if err != nil {
		t.Log(err)
	}
	t.Log(mode)
}

// ListFileNames
func TestListDirFiles(t *testing.T) {
	absPath := fileutil.CurrentPath()

	files, err := fileutil.ListFileNames(absPath)
	if err != nil {
		t.Log(err)
	}

	for _, v := range files {
		t.Log("file: ", v)
	}
}

func TestNetIP(t *testing.T) {
	internalIp := netutil.GetInternalIp()
	ip := net.ParseIP(internalIp)

	t.Log("ip:", ip)

	// ips
	ips := netutil.GetIps()
	t.Log("ips:", ips)

	// out ip
	publicIpInfo, err := netutil.GetPublicIpInfo()
	if err != nil {
		t.Log(err)
	}

	t.Log(publicIpInfo)
}


func TestUserHome(t *testing.T) {
	path, _ := os.UserHomeDir()

	t.Log(path)
}