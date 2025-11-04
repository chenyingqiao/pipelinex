# pipelinex
This is a programming library that can execute CICD tasks and can support Docker, Kubernetes, SSH, and local machines as execution backends.

# 目录

- [x] 流水线图结构推进
- [ ] 流水线执行器支持-Function
- [ ] 流水线执行器支持-Docker
- [ ] 流水线执行器支持-Kubernetes
- [ ] 流水线执行器支持-SSH
- [ ] 流水线执行器支持-Local

# TODO

- status store (现在是和日志强绑定的，需要库中提供接口来实现状态的读取和保存)
- 有向无环图
- k8s ssh持久连接需要一个连接做多个事情，如果这些事情是串行的.
- https://github.com/flosch/pongo2 模板引擎支持，本身就支持多层渲染。 


# 目前问题

- 单元测试无法运行完成
- 配置文件添加状态字段, 但是执行器没有实现根据状态进行执行
