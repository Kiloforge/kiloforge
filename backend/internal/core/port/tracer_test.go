package port

import (
	"context"
	"testing"
)

func TestNoopTracer_StartSpan(t *testing.T) {
	tracer := NoopTracer{}
	ctx, span := tracer.StartSpan(context.Background(), "test-span",
		StringAttr("key", "value"),
		IntAttr("count", 42),
		Float64Attr("cost", 1.5),
	)

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if span == nil {
		t.Fatal("expected non-nil span")
	}

	// All operations should be no-ops without panic.
	span.SetAttributes(StringAttr("foo", "bar"))
	span.AddEvent("test-event", IntAttr("n", 1))
	span.SetError(nil)
	span.End()
}

func TestSpanAttrHelpers(t *testing.T) {
	s := StringAttr("k", "v")
	if s.Key != "k" || s.Value != "v" {
		t.Errorf("StringAttr: got %v", s)
	}

	i := IntAttr("n", 5)
	if i.Key != "n" || i.Value != 5 {
		t.Errorf("IntAttr: got %v", i)
	}

	f := Float64Attr("cost", 3.14)
	if f.Key != "cost" || f.Value != 3.14 {
		t.Errorf("Float64Attr: got %v", f)
	}
}
