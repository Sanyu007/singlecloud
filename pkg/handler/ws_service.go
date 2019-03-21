package handler

import (
	"strings"

	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	WSVersion = resttypes.APIVersion{
		Version: "v1",
		Group:   "ws.zcloud.cn",
	}
)

var (
	ShellClusterPrefix = strings.Join([]string{
		resttypes.GroupPrefix,
		WSVersion.Group,
		WSVersion.Version,
		"clusters",
	}, "/")

	GINShellPath = strings.Join([]string{
		resttypes.GroupPrefix,
		WSVersion.Group,
		WSVersion.Version,
		"clusters",
		":id",
		"*actions",
	}, "/")
)
