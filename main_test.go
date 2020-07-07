package main

import (
	"bytes"
	"testing"
)

func TestInvalidFlags(t *testing.T) {
	flags := []struct {
		args     []string
		expected string
	}{
		{
			[]string{"./noise-test", "--port", "0"},
			`0 is not a valid port number (1-65535)`,
		},
		{
			[]string{"./noise-test", "--port", "65536"},
			`65536 is not a valid port number (1-65535)`,
		},
		{
			[]string{"./noise-test", "--port", "-1"},
			`invalid value "-1" for flag -port: parse error`,
		},
		{
			[]string{"./noise-test", "--remote-port", "0"},
			`0 is not a valid port number (1-65535)`,
		},
		{
			[]string{"./noise-test", "--remote-port", "65536"},
			`65536 is not a valid port number (1-65535)`,
		},
		{
			[]string{"./noise-test", "--remote-port", "-1"},
			`invalid value "-1" for flag -remote-port: parse error`,
		},
		{
			[]string{"./noise-test", "--address", "invalid.address"},
			`invalid.address is not a valid ip address`,
		},
		{
			[]string{"./noise-test", "--address", "127.0.0.300"},
			`127.0.0.300 is not a valid ip address`,
		},
		{
			[]string{"./noise-test", "--remote-address", "invalid.address"},
			`"invalid.address" does not appear to be a valid ip address or server name`,
		},
		{
			[]string{"./noise-test", "--remote-address", "127.0.0.300"},
			`"127.0.0.300" does not appear to be a valid ip address or server name`,
		},
	}
	for _, flag := range flags {
		var stdout, stderr bytes.Buffer
		err := run(flag.args, &stdout, &stderr)

		// all of these tests should error
		if err == nil {
			t.Errorf("%v: error expected, but no error returned", flag.args)
		}

		if err.Error() != flag.expected {
			t.Errorf("%v returned unexpected output.\nExpected: %s\nGot: %s\n", flag.args, flag.expected, err.Error())
		}
	}
}
