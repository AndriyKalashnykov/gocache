package gocache

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestCache_SaveToFile(t *testing.T) {
	defer os.Remove(TestCacheFile)
	cache := NewCache()
	for n := 0; n < 10; n++ {
		cache.Set(strconv.Itoa(n), fmt.Sprintf("v%d", n))
		// To make sure that two entries don't get the exact same timestamp, as that might mess up the order
		time.Sleep(time.Nanosecond)
	}
	err := cache.SaveToFile(TestCacheFile)
	if err != nil {
		t.Fatal("shouldn't have returned an error, but got:", err.Error())
	}
	newCache := NewCache()
	numberOfEntriesEvicted, err := newCache.ReadFromFile(TestCacheFile)
	if err != nil {
		t.Fatal("shouldn't have returned an error, but got:", err.Error())
	}
	if numberOfEntriesEvicted != 0 {
		t.Error("expected 0 entries to have been evicted, but got", numberOfEntriesEvicted)
	}
	if newCache.Count() != 10 {
		t.Error("expected newCache to have 10 entries, but got", newCache.Count())
	}
	if cache.head.Key != newCache.head.Key {
		t.Errorf("head key should've been %s, but was %s", cache.head.Key, newCache.head.Key)
	}
	if cache.tail.Key != newCache.tail.Key {
		t.Errorf("tail key should've been %s, but was %s", cache.tail.Key, newCache.tail.Key)
	}
	if cache.head.previous.Key != newCache.head.previous.Key {
		t.Errorf("head's previous key should've been %s, but was %s", cache.head.previous.Key, newCache.head.previous.Key)
	}
	if cache.tail.next.Key != newCache.tail.next.Key {
		t.Errorf("tail's next key should've been %s, but was %s", cache.tail.next.Key, newCache.tail.next.Key)
	}
}

func TestCache_ReadFromFile(t *testing.T) {
	defer os.Remove(TestCacheFile)
	cache := NewCache()
	for n := 0; n < 10; n++ {
		cache.Set(strconv.Itoa(n), fmt.Sprintf("v%d", n))
		time.Sleep(time.Nanosecond)
	}
	err := cache.SaveToFile(TestCacheFile)
	if err != nil {
		panic(err)
	}
	newCache := cache.WithMaxSize(7)
	numberOfEntriesEvicted, err := newCache.ReadFromFile(TestCacheFile)
	if err != nil {
		panic(err)
	}
	if numberOfEntriesEvicted != 3 {
		t.Error("expected 3 entries to have been evicted, but got", numberOfEntriesEvicted)
	}
	if newCache.Count() != 7 {
		t.Error("expected newCache to have 7 entries since its MaxSize is 7, but got", newCache.Count())
	}
	// Make sure all entries have the right values and can still be GETable
	for key, value := range newCache.entries {
		expectedValue := fmt.Sprintf("v%s", key)
		if value.Value != expectedValue {
			t.Errorf("key %s should've had value '%s', but had '%s' instead", key, expectedValue, value.Value)
		}
		valueFromCacheGet, _ := newCache.Get(key)
		if valueFromCacheGet != expectedValue {
			t.Errorf("key %s should've had value '%s', but had '%s' instead", key, expectedValue, value.Value)
		}
	}
	// Make sure eviction still works
	newCache.evict()
	// Make sure we can create new entries
	newCache.Set("eviction-test", 1)
}
