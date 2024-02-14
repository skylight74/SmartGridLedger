func TestRegisterSession(t *testing.T) {
	// Create a new MockStub
	stub := shimtest.NewMockStub("evcharging", new(EVChargingChaincode))

	// Simulate `registerSession` invocation
	response := stub.MockInvoke("1", [][]byte{
		[]byte("registerSession"),
		[]byte("SESSION123"),
		[]byte("VEHICLE123"),
		[]byte("2021-01-01T12:00:00Z"),
		[]byte("2021-01-01T12:30:00Z"),
		[]byte("10.5"),
	})

	// Check the response
	if response.Status != shim.OK {
		t.Errorf("Expected OK response, got %d", response.Status)
	}

	// Add more checks as needed, for example, querying the session to verify it was stored correctly
}

