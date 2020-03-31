#ifndef _MXOR_H
#define _MXOR_H

// NOTE: Returned data needs to be "malloc"ed so it can be "free"d (don't use "new")

enum MXorCode {MXOR23=0, MXOR24, MXOR26, MXOR27};
// Currently ony MXO26 is implemented

typedef unsigned char Byte;
// The layout of the data pointed at by Byte* is a sequence of one or more of data blocks
// each block being encoded as:
// data length - 4 bytes - big-endian encoding (NUM_DATA_LENGTH_BYTES)
// data bytes - <data length> bytes

#define NUM_DATA_LENGTH_BYTES  4

// Returns the length from the first NUM_DATA_LENGTH_BYTES of "data"
extern unsigned getLength(const Byte* data);
// Sets the length into the first NUM_DATA_LENGTH_BYTES of "data"
void setLength(unsigned len, Byte* buf);

//'data' is a Byte array where the first 4 bytes specify the data length in big endian
// 0-pads the data until the data length is a multiple of the # of shards, encodes the data, then shards it
// Returns shards laid out in contiguous memory in one Byte array, where each shard is a Byte array with the first 4 bytes specifying the data length of the shard in big endian
// If 'data' is NULL or encoding is not yet implenented, returns NULL
extern Byte* mxor_encode(const Byte* data, enum MXorCode mxor_code);

// 'shard1' and 'shard2' are both Byte arrays where the first 4 bytes specify the data length in big endian
// Returns a Byte array of the decoded data, which is the original data that may be 0-padded so that the data length is a multiple of the # of shards.
//  The first 4 bytes of the Byte array specify the data length in big endian.
// If 'shard1' or 'shard2' is NULL or decoding is not yet implemented, returns NULL
extern Byte* mxor_decode(const Byte* shard1, int shard1Ix, const Byte* shard2, int shard2Ix, enum MXorCode mxor_code);

#endif //_MXOR_H
