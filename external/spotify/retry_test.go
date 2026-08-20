package spotify

import (
	"testing"
	"time"
)

func TestRetryAfterWaitRejectsDayLongBan(t *testing.T) {
	wait, retry := retryAfterWait("86400", 0)
	if retry {
		t.Fatalf("retryAfterWait(86400) retry=true wait=%v, want fail-fast", wait)
	}
}

func TestRetryAfterWaitHonorsShortHeader(t *testing.T) {
	wait, retry := retryAfterWait("3", 0)
	if !retry || wait != 3*time.Second {
		t.Fatalf("retryAfterWait(3) = %v, %v, want 3s true", wait, retry)
	}
}
