package oss_svc

//go:generate mockgen -destination=./mock_ossclient/mock.go -package=mock_ossclient github.com/octays/opskat/internal/service/oss_svc Client
