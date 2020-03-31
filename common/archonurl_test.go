package common

import (
	"testing"
)

func TestNewArchonUrl(t *testing.T) {
	toTest := [] struct {
		url			string
		good		bool
		username	string
	}{
		{"arc://marius.eth.n2:6/tmp/dogs.jpg", true, "marius"},
		{"arc://marius.eth.n7:6/tmp/dogs.jpg", false, "marius"},
		{ "arc://h0/HKXZUF6TREGPTV1btM3KkiuXVrtcAiaXDjrPiDAc8Tg7", true, ""},
		{"arc://marius.n2:6/tmp/dogs.jpg", false, "marius"},
		{"arc://marius.xxx.n2:6/tmp/dogs.jpg", false, "marius"},
	}
	for _, cur := range toTest {
		a, err := NewArchonUrl(cur.url)
		isGood := err == nil
		if isGood != cur.good {
			t.Errorf("Validity of %q incorrect, got: %v, want: %v", cur.url, isGood, cur.good)
		} else if cur.good {
			if cur.username != a.Username {
				t.Errorf("%q incorrect username, got: %q, want: %q", cur.url, a.Username, cur.username)
			}
		}
	}
}

func TestShardPath(t *testing.T) {
	toTest := [] struct {
		url       string
		ix			int
		shardPath string
	}{
		{"arc://marius.n2/tmp/Granada.jpg", 2,"marius/tmp/Granada.jpg.2.afs"},
		{"arc://marius.n0/tmp/Granada.jpg", 4,"marius/tmp/Granada.jpg.0.afs"},
		{"arc://narius.h2/68MmsJMtTexAMe9kjx9oQzuAw41cRf7wDFNCEkaDmnxD", 1,  "68MmsJMtTexAMe9kjx9oQzuAw41cRf7wDFNCEkaDmnxD.1.afh"},
		{"arc://marius.h0/68MmsJMtTexAMe9kjx9oQzuAw41cRf7wDFNCEkaDmnxD", 5 , "68MmsJMtTexAMe9kjx9oQzuAw41cRf7wDFNCEkaDmnxD.0.afh"},
	}
	for _, cur := range toTest {
		a, err := NewArchonUrl(cur.url)
		if err != nil {
			t.Error(err)
			continue
		}
		sp := a.ShardPath(cur.ix)
		if sp != cur.shardPath {
			t.Errorf("ShardPath incorrect, got: %q, want: %q", sp, cur.shardPath)
			continue
		}
	}
}
