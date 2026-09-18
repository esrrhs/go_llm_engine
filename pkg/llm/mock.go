package llm

import "context"

// ScriptedClient returns canned responses in order. Used by tests.
type ScriptedClient struct {
	Handle func(ctx context.Context, req Request) (*Response, error)

	Requests []Request
}

func (c *ScriptedClient) Chat(ctx context.Context, req Request) (*Response, error) {
	c.Requests = append(c.Requests, req)
	if c.Handle == nil {
		return nil, nil
	}
	return c.Handle(ctx, req)
}

// SequenceClient walks through a list of responses.
type SequenceClient struct {
	Responses []*Response
	Errs      []error
	I         int
	Requests  []Request
}

func (c *SequenceClient) Chat(ctx context.Context, req Request) (*Response, error) {
	c.Requests = append(c.Requests, req)
	i := c.I
	c.I++
	if i < len(c.Errs) && c.Errs[i] != nil {
		return nil, c.Errs[i]
	}
	if i >= len(c.Responses) {
		return &Response{Content: `{"thought":"no more scripted responses","action":"finish","args":{"summary":"done"}}`}, nil
	}
	return c.Responses[i], nil
}
