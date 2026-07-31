package main

import (
	"context"
	"hash/fnv"

	"github.com/redis/go-redis/v9"
)

// BloomFilter 基于 Redis bitmap（SETBIT/GETBIT）手写的布隆过滤器。
//
// 原理：用 k 个哈希函数把一个元素映射到位数组的 k 个位置并置 1。
// 判断存在时，只要有一个位为 0，就一定不存在；k 个位全为 1，则可能存在
// （存在假阳性，但不存在假阴性）。用在缓存穿透场景刚好：
// 布隆说"不存在"就一定不存在，可以直接拒绝，绝不会漏掉真实数据。
type BloomFilter struct {
	rdb    *redis.Client
	key    string
	bits   uint64 // 位数组长度
	hashes int    // 哈希函数个数 k
}

// NewBloomFilter 构造布隆过滤器。
//   - bits：位数组长度，越大假阳性越低（生产中按预估元素量和目标误判率计算）。
//   - hashes：哈希函数个数 k。
func NewBloomFilter(rdb *redis.Client, key string, bits uint64, hashes int) *BloomFilter {
	return &BloomFilter{rdb: rdb, key: key, bits: bits, hashes: hashes}
}

// offsets 用双哈希（Kirsch-Mitzenmacher）由两个基础哈希组合出 k 个位置，
// 避免维护 k 个独立哈希函数。
func (b *BloomFilter) offsets(id int64) []int64 {
	h := fnv.New64a()
	// 把 int64 id 写成 8 字节喂给哈希。
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(id >> (8 * i))
	}
	h.Write(buf[:])
	sum := h.Sum64()
	h1 := uint32(sum)
	h2 := uint32(sum >> 32)

	res := make([]int64, b.hashes)
	for i := 0; i < b.hashes; i++ {
		combined := h1 + uint32(i)*h2
		res[i] = int64(uint64(combined) % b.bits)
	}
	return res
}

// Add 把 id 加入布隆过滤器：把 k 个位置全部置 1。
func (b *BloomFilter) Add(ctx context.Context, id int64) error {
	pipe := b.rdb.Pipeline()
	for _, off := range b.offsets(id) {
		pipe.SetBit(ctx, b.key, off, 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// MightContain 判断 id 是否可能存在：k 个位全为 1 才返回 true。
func (b *BloomFilter) MightContain(ctx context.Context, id int64) (bool, error) {
	offs := b.offsets(id)
	pipe := b.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(offs))
	for i, off := range offs {
		cmds[i] = pipe.GetBit(ctx, b.key, off)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	for _, cmd := range cmds {
		if cmd.Val() == 0 {
			return false, nil // 有一位是 0，一定不存在
		}
	}
	return true, nil // 全 1，可能存在（有假阳性）
}
