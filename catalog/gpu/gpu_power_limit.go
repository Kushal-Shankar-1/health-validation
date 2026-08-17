package gpu

import "github.com/dsx-ai-factory/health-validation/catalog/health"

// GPUPowerLimit returns the check definition that verifies each GPU's
// current and enforced power limits.
//
// Operator thresholds are configured in watts (expected_power_limit_w).
// Observed signals are NVML milliwatts (power_*_mw); CEL compares
// signal == watts*1000. Limits must match expected_power_limit_w, which
// defaults per architecture (operators may override). The catalog Value is
// a placeholder (GB200's 1200 W) used only when architecture filtering is
// off; declared arches resolve via PerArchitecture. Readings are compared
// exactly in milliwatts so anomalous fractional-watt values do not round
// into a pass.
func GPUPowerLimit() health.CheckDefinition {
	return health.CheckDefinition{
		ID:          "gpu_power_limit",
		Name:        "GPU Power Limit",
		Description: "Verifies each GPU's current and enforced power limits.",
		Category:    health.GPU,

		RequiredSignals: []string{
			SignalGPUID,
			SignalPowerLimitMW,
			SignalEnforcedPowerLimitMW,
		},

		ExecutionClass:    health.Passive,
		ExecutionContexts: []health.ExecutionContext{health.OperatorInfra, health.TenantGuest},
		SchedulingCadence: health.Prolog,
		Architectures:     []string{"GB200", "GB300", "H100"},

		Provenance:        health.NVIDIA,
		MaturityLevel:     health.Stable,
		RecommendedAction: health.Drain(health.DiagnosticActionCheckMechanicals),

		Thresholds: map[string]health.ThresholdSpec{
			"expected_power_limit_w": {
				// Placeholder when architecture filtering is off. Declared
				// arches resolve via PerArchitecture (GB200 1200 W).
				Value: int64(1200),
				PerArchitecture: map[string]any{
					"GB200": int64(1200),
					"GB300": int64(1400),
					"H100":  int64(700),
				},
				Category:    health.DefaultThreshold,
				Description: "Current and enforced power limit expected for every GPU.",
				Unit:        "W",
			},
		},

		// Thresholds are watts; signals are NVML milliwatts. Compare exactly
		// via watts*1000 — rounding mW→W could hide anomalous fractional limits.
		Rules: []health.Rule{
			{
				Expression: "power_limit_mw != expected_power_limit_w * 1000",
				Severity:   health.Critical,
				Message:    "GPU ${gpu_id}: current power limit ${power_limit_mw} mW, expected ${expected_power_limit_w} W.",
			},
			{
				Expression: "enforced_power_limit_mw != expected_power_limit_w * 1000",
				Severity:   health.Critical,
				Message:    "GPU ${gpu_id}: enforced power limit ${enforced_power_limit_mw} mW, expected ${expected_power_limit_w} W.",
			},
		},
	}
}
