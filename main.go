package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/14rcole/distbuilder/worker/builder"
	"github.com/14rcole/distbuilder/worker/networking"
	"github.com/sirupsen/logrus"
)

type BuildRequest struct {
	Success bool   `json:"success"` // if this is true, error will be "", else, diff will be ""
	Diff    []byte `json:"diff"`
	Error   string `json:"error"`
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	logrus.Debug("Testing build...")
	// Create and populate parallelbuilder
	pb, err := mockBuilder()
	if err != nil {
		logrus.Fatal(err)
	}
	pb.Options.ReportWriter = os.Stdout

	body, err := pb.MarshalJSON()
	if err != nil {
		logrus.Fatal(err)
	}
	req, err := http.NewRequest("GET", "/", bytes.NewReader(body))
	if err != nil {
		logrus.Fatal(err)
	}

	// We create a responseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(networking.BuildHandler)

	logrus.Debug("Sending request...")
	// We can call the handler's ServeHTTP method directly and pass in our
	// Request and ResponseRecorder
	handler.ServeHTTP(rr, req)

	// Check that the status code is what we expect
	if status := rr.Code; status != http.StatusOK {
		logrus.Warnf("handler returned wrong status code: got %v wanted %v", status, http.StatusOK)
	}

	// Check that the response body is what we expect
	// convert the body to JSON
	body = []byte(rr.Body.String())
	results := new(BuildRequest)
	err = json.Unmarshal(body, results)
	if err != nil {
		logrus.Debugf("Body: %q", body)
		logrus.Warnf("%s", err)
	}
	if !results.Success {
		logrus.Warnf("Could not build container: %q", results.Error)
	}
}

func mockBuilder() (*builder.Builder, error) {
	buildString := `{
		"Builder": {
		  "AllowedArgs": {
			"FTP_PROXY": true,
			"HTTPS_PROXY": true,
			"HTTP_PROXY": true,
			"NO_PROXY": true,
			"ftp_proxy": true,
			"http_proxy": true,
			"https_proxy": true,
			"no_proxy": true
		  },
		  "Args": {},
		  "Author": "",
		  "CmdSet": false,
		  "Env": [
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
		  ],
		  "Excludes": null,
		  "PendingCopies": [],
		  "PendingRuns": [],
		  "PendingVolumes": null,
		  "RunConfig": {
			"ArgsEscaped": false,
			"AttachStderr": false,
			"AttachStdin": false,
			"AttachStdout": false,
			"CPUSet": "",
			"CPUShares": 0,
			"Cmd": [
			  "/bin/sh"
			],
			"DNS": null,
			"Domainname": "",
			"Entrypoint": [],
			"Env": null,
			"ExposedPorts": {},
			"Healthcheck": null,
			"Hostname": "",
			"Image": "alpine",
			"KernelMemory": 0,
			"Labels": {},
			"MacAddress": "",
			"Memory": 0,
			"MemoryReservation": 0,
			"MemorySwap": 0,
			"Mounts": [],
			"NetworkDisabled": false,
			"OnBuild": null,
			"OpenStdin": false,
			"PortSpecs": null,
			"PublishService": "",
			"SecurityOpts": null,
			"Shell": null,
			"StdinOnce": false,
			"StopSignal": "",
			"StopTimeout": 0,
			"Tty": false,
			"User": "",
			"VolumeDriver": "",
			"Volumes": {},
			"VolumesFrom": "",
			"WorkingDir": "/"
		  },
		  "Volumes": null,
		  "Warnings": null
		},
		"Diff": null,
		"From": "alpine",
		"Node": null,
		"Options": {
		  "AdditionalTags": [],
		  "Args": {},
		  "CommonBuildOpts": {
			"AddHost": [],
			"ApparmorProfile": "",
			"CPUPeriod": 0,
			"CPUQuota": 0,
			"CPUSetCPUs": "",
			"CPUSetMems": "",
			"CPUShares": 0,
			"CgroupParent": "",
			"LabelOpts": null,
			"Memory": 0,
			"MemorySwap": 0,
			"SeccompProfilePath": "",
			"ShmSize": "65536k",
			"Ulimit": [],
			"Volumes": []
		  },
		  "Compression": 2,
		  "ContextDirectory": "/home/ryan/Development/Go/src/github.com/projectatomic/buildah/tests/bud/shell",
		  "DefaultMountsFilePath": "/usr/share/containers/mounts.conf",
		  "Err": null,
		  "IgnoreUnrecognizedInstructions": false,
		  "Out": null,
		  "Output": "shell-test",
		  "OutputFormat": "application/vnd.oci.image.manifest.v1+json",
		  "Parallel": true,
		  "PullPolicy": 0,
		  "Quiet": false,
		  "Registry": "",
		  "Runtime": "runc",
		  "RuntimeArgs": [],
		  "SignaturePolicyPath": "",
		  "SystemContext": {
			"ArchitectureChoice": "",
			"AuthFilePath": "",
			"DirForceCompress": false,
			"DockerAuthConfig": null,
			"DockerCertPath": "",
			"DockerDaemonCertPath": "",
			"DockerDaemonHost": "",
			"DockerDaemonInsecureSkipTLSVerify": false,
			"DockerDisableV1Ping": false,
			"DockerInsecureSkipTLSVerify": false,
			"DockerPerHostCertDirPath": "",
			"DockerRegistryUserAgent": "",
			"OCICertPath": "",
			"OCIInsecureSkipTLSVerify": false,
			"OCISharedBlobDirPath": "",
			"OSChoice": "",
			"OSTreeTmpDirPath": "",
			"RegistriesDirPath": "",
			"RootForImplicitAbsolutePaths": "",
			"SignaturePolicyPath": "",
			"SystemRegistriesConfPath": ""
		  },
		  "TransientMounts": [],
		  "Transport": "",
		  "Workers": [
			"localhost"
		  ]
		}
		}	  
	  `

	pb := new(builder.Builder)
	err := json.Unmarshal([]byte(buildString), pb)
	return pb, err
}
