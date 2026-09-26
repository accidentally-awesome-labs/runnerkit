package bootstrap

import "fmt"

const RunnerVersion = "2.337.0"

type RunnerPackage struct {
	Version  string
	OS       string
	Arch     string
	Filename string
	URL      string
	SHA256   string
}

func PackageFor(osName string, arch string) (RunnerPackage, error) {
	switch osName + "/" + arch {
	case "linux/x64":
		return RunnerPackage{
			Version:  RunnerVersion,
			OS:       "linux",
			Arch:     "x64",
			Filename: "actions-runner-linux-x64-2.337.0.tar.gz",
			URL:      "https://github.com/actions/runner/releases/download/v2.337.0/actions-runner-linux-x64-2.337.0.tar.gz",
			SHA256:   "70920811a4f8ad4328818682bca5c6469c1c942fab52448868071d0063816613",
		}, nil
	case "linux/arm64":
		return RunnerPackage{
			Version:  RunnerVersion,
			OS:       "linux",
			Arch:     "arm64",
			Filename: "actions-runner-linux-arm64-2.337.0.tar.gz",
			URL:      "https://github.com/actions/runner/releases/download/v2.337.0/actions-runner-linux-arm64-2.337.0.tar.gz",
			SHA256:   "9b1dc70626422526e3c94767cf024896beb15da5342a3f4819bf2feac13e0393",
		}, nil
	default:
		return RunnerPackage{}, fmt.Errorf("unsupported runner package; supported packages are linux/x64 and linux/arm64")
	}
}
