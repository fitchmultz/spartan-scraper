package store

// ListOptions specifies pagination parameters for Store.ListOpts.
type ListOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListOptions) Defaults() ListOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}

// ListByStatusOptions specifies pagination parameters for Store.ListByStatus.
type ListByStatusOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListByStatusOptions) Defaults() ListByStatusOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}

// ListCrawlStatesOptions specifies pagination parameters for Store.ListCrawlStates.
type ListCrawlStatesOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListCrawlStatesOptions) Defaults() ListCrawlStatesOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}
