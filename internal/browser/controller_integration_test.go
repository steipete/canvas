//go:build integration

package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTypeClear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><input id="input" value="old"><textarea id="textarea">old</textarea>
<script>
window.changes = [];
document.addEventListener('change', event => changes.push(event.target.value));
</script>`))
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	controller, err := New(ctx, Options{Headless: true, UserDataDir: t.TempDir(), StartURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = controller.Close() })
	if _, _, err := controller.Navigate(ctx, server.URL); err != nil {
		t.Fatal(err)
	}
	for _, selector := range []string{"#input", "#textarea"} {
		t.Run(selector, func(t *testing.T) {
			for _, step := range []struct {
				text  string
				clear bool
				want  string
			}{
				{text: "replacement", clear: true, want: "replacement"},
				{text: "+suffix", want: "replacement+suffix"},
				{clear: true, want: ""},
			} {
				if err := controller.Type(ctx, selector, step.text, step.clear); err != nil {
					t.Fatal(err)
				}
				got, err := controller.Eval(ctx, `document.querySelector('`+selector+`').value`)
				if err != nil {
					t.Fatal(err)
				}
				if got != step.want {
					t.Fatalf("value = %q, want %q", got, step.want)
				}
			}
		})
	}
	got, err := controller.Eval(ctx, `changes.filter(value => value === '').length >= 4`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Fatal("clearing must dispatch change events with an empty value")
	}
}
