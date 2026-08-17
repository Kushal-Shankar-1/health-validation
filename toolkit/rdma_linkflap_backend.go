package toolkit

import (
	"context"

	"github.com/dsx-ai-factory/health-validation/catalog/health"
	"github.com/dsx-ai-factory/health-validation/catalog/infiniband"
)

// linkFlapGrep is the journalctl --grep prefilter for mlx5 link
// transitions. It is deliberately looser than the source check's
// `mlx5_core .* rdma.*: Link (up|down)` — that pattern required an "rdma"
// token the driver's netdev link lines do not carry — so real outages
// are collected; the parser does the exact structuring.
const linkFlapGrep = `(?i)mlx5_core .* Link (up|down)`

// RDMALinkFlapBackend is a single data source for mlx5 RDMA link-flap
// events. The production backend reads the system journal; tests inject
// canned log lines via [MockRDMALinkFlap]. It returns one aggregate
// [infiniband.LinkFlapMeasurement] (even when no flaps were found)
// wrapped in [health.Measurement], so the catalog always has a
// measurement to evaluate.
type RDMALinkFlapBackend interface {
	Measure(ctx context.Context) ([]health.Measurement, error)
}

// JournalctlRDMALinkFlap returns a backend that collects mlx5 RDMA link
// transitions from the current boot's kernel journal by running:
//
//	journalctl --dmesg --boot --no-pager --output=short-full --grep=<mlx5 link>
//
// --output=short-full is required (not --output=cat) because the parser
// needs the per-line timestamp to measure outage duration. Requires
// journalctl on PATH and permission to read the system journal. A read
// failure (or timeout) yields a single errored measurement rather than an
// error from Measure, so the catalog fails the link-flap check instead of
// dropping the host.
func JournalctlRDMALinkFlap() RDMALinkFlapBackend {
	return &rdmaLinkFlapBackend{run: runJournalctl}
}

// rdmaLinkFlapBackend reads link lines from the journal. run is a seam so
// tests can exercise parsing and error handling without a host journal.
type rdmaLinkFlapBackend struct {
	run runFunc
}

// Measure reads the current boot's mlx5 link transitions. --output=short-full
// is required (not --output=cat) because the parser needs the per-line
// timestamp to measure outage duration. See [readJournal] for the clean-boot
// vs unreadable-journal handling.
func (b *rdmaLinkFlapBackend) Measure(ctx context.Context) ([]health.Measurement, error) {
	return readJournal(ctx, b.run, journalMatcher{
		args:    []string{"--output=short-full", "--grep=" + linkFlapGrep},
		subject: "link state",
		parse:   func(lines []string) health.Measurement { return ParseLinkFlaps(sinceBoot, lines) },
		errored: func(err error) health.Measurement {
			return infiniband.ErroredLinkFlap(sinceBoot, err, infiniband.MeasurementSignals...)
		},
	})
}

// MockRDMALinkFlap returns a backend that parses the provided short-full
// log lines with the real parser, as if journalctl had returned them.
// Pass no lines to simulate a flap-free boot.
//
//	sensor := toolkit.NewRDMALinkFlapSensor(toolkit.MockRDMALinkFlap(
//	    "Thu 2025-12-11 11:49:00 UTC host kernel: mlx5_core 0000:5a:00.0 enp90s0np0: Link down",
//	    "Thu 2025-12-11 11:51:00 UTC host kernel: mlx5_core 0000:5a:00.0 enp90s0np0: Link up",
//	))
func MockRDMALinkFlap(lines ...string) RDMALinkFlapBackend {
	return &mockRDMALinkFlapBackend{lines: lines}
}

type mockRDMALinkFlapBackend struct {
	lines []string
}

func (m *mockRDMALinkFlapBackend) Measure(context.Context) ([]health.Measurement, error) {
	return []health.Measurement{ParseLinkFlaps(sinceBoot, m.lines)}, nil
}
