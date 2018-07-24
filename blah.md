
# What I've done

I did an analysis on some series files. The index is not a significant portion of the space. Over 90% of it is in easily compressible segment files. I have come up with a backwards compatible proposal to write new segement files with lz4 compression, which is optimized for very fast decompression and cheap, good compression on easily compressible data. Additionally, I think there are some possible wins for series file index compaction if we include another 8 bytes per element.

I have a branch pushed with some work on compressing the segment files. It will be able to read old files but only write new files with compression.

# Long stream of consiousness spew with details

#### Stats per partition
- the index is 32MB
- the segments are 4 + 8 + 16 + 32 + 64 + 128 = 252MB
- which makes the index 11% of the data
- lz4 compresses the segments down to 17.5MB or 7% of the original
- lz4 compression happens in 250ms
- lz4 decompression is 330ms
- these times include reading/writing to disk
- i believe the disk is the bottleneck for those tests

#### about the index
- the index compresses from 32MB to 17MB
- it's fairly dense. there are quite a bit of null bytes, but they're interspersed
- this makes me think the load factor is close to the targeted 90%
- we COULD waste up to a factor of 2 thanks to compaction
- i think this cost is going to be dwarfed by the segments over time
- since index compaction removes tombstones, it's going to be bounded by the amount of active series
- the logs grow forever, and are nearly guaranteed to have a waste factor of 2 due to the way we grow them

#### thoughts/compression proposal
- i really think compression on the segment files is going to be a win
- the hard part about this is providing O(1) access to log entries recorded in the index
- currently the offsets are uint64s that actually only encode 32+16 bits of information
- specifically a 16 bit segment id and a 32 bit offset inside of that segment

```
┌───────┬───────┬───────────────┐
│       | Seg   | Offset        |
├───┬───┼───┬───┼───┬───┬───┬───┤
│ 7 │ 6 | 5 | 4 | 3 | 2 | 1 | 0 | byte index
└───┴───┴───┴───┴───┴───┴───┴───┘

where Seg is the segment number
and Offset is the byte offset into the segment
```

- so we know the two highest order (6 and 7) bytes are zero
- additionally, each time a segment number goes up, it's size doubles, eventually hitting 256MB
- there are 6 files before it starts hitting 256
- 16 bits there provides 16TB of segment files
- using 8 bits instead provides 62.75GB
- that's 500GB for 8 partitions
- i think that amount should be sufficient for long enough to implement compaction lol
- reducing that allows this kind of layout

```
┌───────────┬───┬───────────────┐
│ Index     | S | Offset        |
├───┬───────┼───┼───┬───┬───┬───┤
│ 7 │ 6 | 5 | 4 | 3 | 2 | 1 | 0 | byte index
└───┴───┴───┴───┴───┴───┴───┴───┘

where S is the segment number
and Offset is the byte offset into the segment
and Index is a 23 bit integer of a byte offset into a decompressed block
```

- the highest order bit is used to flag that the offset is in a compressed block
- we make index 23 bits to flag the highest bit as one containing an index
- offset remains the offset into a compressed segment file
- it will be shared by a number of entries that have been compressed into a block
- the index is the byte index inside of the decompressed block
- this allows us to compress up to 8MB of data at once
- we could instead/additionally reduce offset from 32 to 28 bits since segments are at most 256MB anyway and add that to either the index or the segment. we have essentially 64 - 1 - 28 = 35 bits to distribute between the segment number and the index
- when attempting to access a record at some offset we would:
    - seek to the compressed block by offset
    - read a 4 byte value encoding the length of the compressed block
    - read a 4 byte value encoding the length of the uncompressed block
    - decompress the block
    - seek into the decompressed block by the index value
    - return the byte slice and/or append the value into a provided buffer
- right now, we just return a byte slice to the mmap'd region
- under this proposal we would have to allocate/copy the data out
- this can be mitigated with some caching because a significant portion of the traffic
- i suspect the most expensive part of that is page faulting when the caller attempts to read
- i have profiles that back that assertion up. a large time is spent in encoding/binary.Uvarint
- that only makes sense if it's reading faulting data.
- lz4 is stupid fast and optimized for decoding reads
- we might actually see performance wins during index compaction because of disk traffic reductions

## tangent series file index compaction thought

during compaction, we do the RHH probing to see if the existing element is probed less than us. if so, we swap with existing and then try to find a better spot for them. in order to check that, we read in the key from the segments. this is essentially a random read from them, almost guaranteed to fault. it might be worthwhile to bloat the index some by including the hash of the key in the index entry. that would mean that we wouldn't have to consult the segments, and would piggy back on the read we already had to do to see if the slot was empty. this is consistently the largest cumulative portion in the profiles.


