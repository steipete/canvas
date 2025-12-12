package cmd

import "testing"

func TestStartCmd_StealthFlagDefault(t *testing.T) {
	flags := &rootFlags{}
	cmd := newStartCmd(flags)
	v, err := cmd.Flags().GetBool("stealth")
	if err != nil {
		t.Fatal(err)
	}
	if !v {
		t.Fatalf("expected --stealth default true")
	}
}

