package service

import (
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func isInvalidAddress(addr string) bool {
	return addr == "" || addr == "<nil>"
}

func GetCallbackAddress() string {
	addr := operation_setting.CustomCallbackAddress
	if isInvalidAddress(addr) {
		addr = system_setting.ServerAddress
	}
	if isInvalidAddress(addr) {
		return ""
	}
	return addr
}

func GetServerAddress() string {
	addr := system_setting.ServerAddress
	if isInvalidAddress(addr) {
		return ""
	}
	return addr
}
