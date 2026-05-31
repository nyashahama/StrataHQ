package dbgen

import (
	"strings"
	"testing"
)

func TestUpdateEarlyAccessStatusRequiresPendingStatus(t *testing.T) {
	if !strings.Contains(updateEarlyAccessStatus, "AND status = 'pending'") {
		t.Fatalf("UpdateEarlyAccessStatus must only transition pending rows:\n%s", updateEarlyAccessStatus)
	}
}
