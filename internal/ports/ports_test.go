package ports

import (
	"testing"
)

func TestRealPortChecker_IsAvailable(t *testing.T) {
	ResetCache()
	checker := RealPortChecker{}

	// Test with a port that should be available (high port number)
	// Port 0 is invalid, but we test with a high port that's unlikely to be used
	available := checker.IsAvailable(54321)
	if !available {
		t.Errorf("Expected port 54321 to be available")
	}
}

func TestRealPortChecker_IsAvailable_InvalidPort(t *testing.T) {
	checker := RealPortChecker{}

	// Test invalid ports
	if checker.IsAvailable(0) {
		t.Errorf("Expected port 0 to be invalid")
	}
	if checker.IsAvailable(-1) {
		t.Errorf("Expected port -1 to be invalid")
	}
	if checker.IsAvailable(65536) {
		t.Errorf("Expected port 65536 to be invalid")
	}
}

func TestFindAvailablePort(t *testing.T) {
	ResetCache()
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			8080: false,
			8081: true,
			8082: true,
		},
	}

	port := FindAvailablePort(mock, 8080, 10)
	if port != 8081 {
		t.Errorf("Expected port 8081, got %d", port)
	}
}

func TestFindAvailablePort_StartPortAvailable(t *testing.T) {
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			8080: true,
		},
	}

	port := FindAvailablePort(mock, 8080, 10)
	if port != 8080 {
		t.Errorf("Expected port 8080, got %d", port)
	}
}

func TestFindAvailablePort_MultipleOccupied(t *testing.T) {
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			8080: false,
			8081: false,
			8082: false,
			8083: true,
		},
	}

	port := FindAvailablePort(mock, 8080, 10)
	if port != 8083 {
		t.Errorf("Expected port 8083, got %d", port)
	}
}

func TestFindAvailablePort_PostgreSQL(t *testing.T) {
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			5432: false,
			5433: true,
		},
	}

	port := FindAvailablePort(mock, 5432, 10)
	if port != 5433 {
		t.Errorf("Expected port 5433, got %d", port)
	}
}

func TestMockPortChecker(t *testing.T) {
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			8080: true,
			9000: false,
		},
	}

	if !mock.IsAvailable(8080) {
		t.Errorf("Expected port 8080 to be available")
	}
	if mock.IsAvailable(9000) {
		t.Errorf("Expected port 9000 to be unavailable")
	}
	// Default is available
	if !mock.IsAvailable(12345) {
		t.Errorf("Expected port 12345 to be available by default")
	}
}

func TestFindAvailablePort_Fallback(t *testing.T) {
	mock := MockPortChecker{
		AvailablePorts: map[int]bool{
			8080: false,
			8081: false,
			8082: false,
		},
	}

	// With maxAttempts = 3, all are occupied, should fallback to startPort
	port := FindAvailablePort(mock, 8080, 3)
	if port != 8080 {
		t.Errorf("Expected fallback to 8080, got %d", port)
	}

}
