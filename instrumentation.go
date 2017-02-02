package main

// InstrumentationTable holds points to various structures making them available for instrumentation.
type InstrumentationTable struct {
	connectionTable *ConnectionTable
}

func newInstrumentationTable(connectionTable *ConnectionTable) *InstrumentationTable {
	return &InstrumentationTable{
		connectionTable: connectionTable,
	}
}
