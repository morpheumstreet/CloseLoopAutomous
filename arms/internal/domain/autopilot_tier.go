package domain

import (
	"fmt"
	"strings"
)

// AutomationTier controls how aggressively the autopilot runs (Mission Control–style tiers).
type AutomationTier string

const (
	TierSupervised AutomationTier = "supervised" // human gates each major step; cadences optional nudges only
	TierSemiAuto   AutomationTier = "semi_auto"  // scheduled research/ideation; dispatch still manual by default
	TierFullAuto   AutomationTier = "full_auto"  // scheduled phases + auto_dispatch_enabled may dispatch (when wired)
)

func ParseAutomationTier(s string) (AutomationTier, error) {
	t := AutomationTier(strings.ToLower(strings.TrimSpace(s)))
	switch t {
	case TierSupervised, TierSemiAuto, TierFullAuto:
		return t, nil
	case "":
		return TierSupervised, nil
	default:
		return "", fmt.Errorf("unknown automation_tier %q (want supervised, semi_auto, full_auto)", s)
	}
}

func (t AutomationTier) String() string { return string(t) }
