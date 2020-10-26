package gocache

import (
	"bufio"
	"encoding/gob"
	"os"
	"sort"
)

// SaveToFile stores the content of the cache to a file so that it can be read using
// the ReadFromFile function
func (cache *Cache) SaveToFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	encoder := gob.NewEncoder(writer)
	cache.mutex.RLock()
	err = encoder.Encode(cache.entries)
	cache.mutex.RUnlock()
	if err != nil {
		return err
	}
	return writer.Flush()
}

// ReadFromFile populates the cache using a file created using cache.SaveToFile(path)
//
// Note that if the number of entries retrieved from the file exceed the configured maxSize,
// the extra entries will be automatically evicted according to the EvictionPolicy configured.
// This function returns the number of entries evicted, and because this function only reads
// from a file and does not modify it, you can safely retry this function after configuring
// the cache with the appropriate maxSize, should you desire to.
func (cache *Cache) ReadFromFile(path string) (int, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	decoder := gob.NewDecoder(reader)
	cache.mutex.Lock()
	err = decoder.Decode(&cache.entries)
	if err != nil {
		return 0, err
	}
	// Because pointers don't get stored in the file, we need to relink everything from head to tail
	var entries []*Entry
	for _, v := range cache.entries {
		entries = append(entries, v)
	}
	// Sort the slice of entries from oldest to newest
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelevantTimestamp.Before(entries[j].RelevantTimestamp)
	})
	// Relink the nodes from tail to head
	var previous *Entry
	for i := range entries {
		current := entries[i]
		if previous == nil {
			cache.tail = current
			cache.head = current
		} else {
			previous.next = current
			current.previous = previous
			cache.head = current
		}
		previous = entries[i]
		if cache.maxMemoryUsage != NoMaxMemoryUsage {
			cache.memoryUsage += current.SizeInBytes()
		}
	}
	// If the cache doesn't have a maxSize/maxMemoryUsage, then there's no point checking if we need to evict
	// an entry, so we'll just return now
	if cache.maxSize == NoMaxSize && cache.maxMemoryUsage == NoMaxMemoryUsage {
		cache.mutex.Unlock()
		return 0, nil
	}
	// Evict what needs to be evicted
	numberOfEvictions := 0
	// If there's a maxSize and the cache has more entries than the maxSize, evict
	if cache.maxSize != NoMaxSize && len(cache.entries) > cache.maxSize {
		for len(cache.entries) > cache.maxSize {
			numberOfEvictions++
			cache.evict()
		}
	}
	// If there's a maxMemoryUsage and the memoryUsage is above the maxMemoryUsage, evict
	if cache.maxMemoryUsage != NoMaxMemoryUsage && cache.memoryUsage > cache.maxMemoryUsage {
		for cache.memoryUsage > cache.maxMemoryUsage && len(cache.entries) > 0 {
			numberOfEvictions++
			cache.evict()
		}
	}
	cache.mutex.Unlock()
	return numberOfEvictions, nil
}
