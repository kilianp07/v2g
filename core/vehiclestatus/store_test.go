package vehiclestatus

import "testing"

func TestMemoryStore_Filter(t *testing.T) {
	s := NewMemoryStore()
	s.Set(Status{VehicleID: "v1", FleetID: "f1", Site: "s1"})
	s.Set(Status{VehicleID: "v2", FleetID: "f2", Site: "s2"})
	out := s.List(Filter{FleetID: "f1"})
	if len(out) != 1 || out[0].VehicleID != "v1" {
		t.Fatalf("filter failed: %#v", out)
	}
}

func TestMemoryStore_FilterCluster(t *testing.T) {
	s := NewMemoryStore()
	s.Set(Status{VehicleID: "v1", Cluster: "c1"})
	s.Set(Status{VehicleID: "v2", Cluster: "c2"})
	out := s.List(Filter{Cluster: "c2"})
	if len(out) != 1 || out[0].VehicleID != "v2" {
		t.Fatalf("cluster filter failed: %#v", out)
	}
}

func TestMemoryStore_RecordDispatch(t *testing.T) {
	s := NewMemoryStore()
	s.Set(Status{VehicleID: "v1"})
	dec := LastDispatch{SignalType: "FCR", TargetPower: 1}
	s.RecordDispatch("v1", dec)
	out := s.List(Filter{})
	if out[0].CurrentStatus != "dispatched" {
		t.Fatalf("status not updated")
	}
}

func TestMemoryStore_RecordDispatchNew(t *testing.T) {
	s := NewMemoryStore()
	dec := LastDispatch{SignalType: "FCR"}
	s.RecordDispatch("v3", dec)
	out := s.List(Filter{})
	if len(out) != 1 || out[0].VehicleID != "v3" {
		t.Fatalf("auto create failed %#v", out)
	}
}
