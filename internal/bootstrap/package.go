package bootstrap

import "fmt"

const RunnerVersion = "2.334.0"

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
			Filename: "actions-runner-linux-x64-2.334.0.tar.gz",
			URL:      "https://github.com/actions/runner/releases/download/v2.334.0/actions-runner-linux-x64-2.334.0.tar.gz",
			SHA256:   "048024cd2c848eb6f14d5646d56c13a4def2ae7ee3ad12122bee960c56f3d271",
		}, nil
	case "linux/arm64":
		return RunnerPackage{
			Version:  RunnerVersion,
			OS:       "linux",
			Arch:     "arm64",
			Filename: "actions-runner-linux-arm64-2.334.0.tar.gz",
			URL:      "https://github.com/actions/runner/releases/download/v2.334.0/actions-runner-linux-arm64-2.334.0.tar.gz",
			SHA256:   "f44255bd3e80160eb25f71bc83d06ea025f6908748807a584687b3184759f7e4",
		}, nil
	default:
		return RunnerPackage{}, fmt.Errorf("unsupported runner package; supported packages are linux/x64 and linux/arm64")
	}
}
