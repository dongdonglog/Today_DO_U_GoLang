# 第七阶段 Go 高并发与性能

## 目标

成为高级开发工程师

## 占比

10%

## 章节列表

| 编号 | 章节 |
|------|------|
| 48 | [Goroutine](./48-Goroutine.md) |
| 49 | [Channel](./49-Channel.md) |
| 50 | [Mutex 与 RWMutex](./50-Mutex与RWMutex.md) |
| 51 | [原子操作](./51-原子操作.md) |
| 52 | [并发模式](./52-并发模式.md) |
| 53 | [Goroutine 泄漏](./53-Goroutine泄漏.md) |
| 54 | [pprof 性能分析](./54-pprof性能分析.md) |
| 55 | [Benchmark 测试](./55-Benchmark测试.md) |

## 阶段项目

**秒杀系统**

## 技术栈

- Goroutine
- Channel
- sync包
- pprof
- Benchmark

## 学习顺序

先用有界 Goroutine 和 Channel 建立任务生命周期，再用锁和原子操作保护共享状态；随后组合 worker pool、取消与背压，最后用 pprof 和 Benchmark 根据数据定位瓶颈。章节代码使用 Go 标准库，可独立保存为示例文件运行。
