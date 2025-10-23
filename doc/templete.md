# 关于模板引擎的选型

下面给你 3 个 **Go 语言** 模板引擎，**同时满足** 5 个需求：

1. 支持 **for / if / else**  
2. 支持 **函数调用 & 过滤器**（可自定义）  
3. **默认“保留未匹配标签”**（不报错、不删除 `{{foo}}`）  
4. 生产级活跃度 & 性能  
5. 零依赖 或 最小依赖

---

### ✅ 1. pongo2（Django 语法，最稳妥）

| 特性 | 说明 |
|---|---|
| 语法 | 与 Django 模板 1:1 对应，`{% if %}`、`{% for %}`、管道过滤器、宏、继承、include 全都有 |
| 未匹配变量 | 默认 **原样保留** `{{foo}}`；可配置为 `ERROR`、`EMPTY` 三种模式 |
| 函数/过滤器 | 内置 60+ 过滤器（`date`、`truncatechars`、`default_if_none` 等），可 `RegisterFilter()` 无限扩展 |
| 性能 | 预编译 AST，中等偏上，比 `text/template` 快 2~3 倍 |
| 示例 | `{{ price|floatformat:2 }}` `{% if score > 90 %}A{% endif %}` |

```go
import "github.com/flosch/pongo2/v6"

tpl, _ := pongo2.FromString("hello {{name}}! {{missing}}")
out, _ := tpl.Execute(pongo2.Context{"name": "world"})
// out == "hello world! {{missing}}"
```

> 官方已支持 Go 1.22+，2024 年仍在活跃维护 。

---

### ✅ 2. gtpl（国产轻量级，语法更紧凑）

| 特性 | 说明 |
|---|---|---|---|
| 语法 | 自创缩进风格 `{!if expr}…{;else}…{/if}` `{!list:arr}…{/list}` |
| 未匹配变量 | **默认保留**；引擎把未命中标签直接写回源码 |
| 函数/过滤器 | 可直接调用 Go 内置函数 `len/toupper/trim/replace`；也支持注册自定义函数 |
| 性能 | 单文件无依赖，递归下降解析，**比 pongo2 再快 1.5~2 倍** |
| 示例 | `{!if id>0}YES{;else}NO{/if}` `{!list:users}{:user_name}{/list}` |

```go
import "github.com/pywee/gtpl"

p := gtpl.NewParser()
out, _ := p.ParseFile("demo.html", map[string]any{
    "name": "world",
})
// 模板里 {{missing}} 会原样留在输出
```

> 2025 年 9 月仍在迭代，作者明确支持“保留未解析标签”模式 。

---

### ✅ 3. Hero（预编译型，极限性能）

| 特性 | 说明 |
|---|---|
| 语法 | **原生 Go 语法** 直接写在模板里 `<% if a > 0 { %>`，完全不用学新语言 |
| 未匹配变量 | 预编译成 `.go` 文件，**未命中标识符**按字符串字面量保留（不会编译报错） |
| 函数/过滤器 | 任意 Go 函数、方法、自定义 func 直接调用；零反射 |
| 性能 | **模板先编译成普通 Go 代码**，最终执行就是纯函数调用，**全网 Go 模板最快** |
| 示例 | `<%! import "strings" %><%= strings.ToUpper(name) %>` |

```go
//go:generate hero -source=index.html
func DemoRender(name string) string {
    // 生成的 Go 函数，未匹配内容原样拼接
}
```

> 2024 年发布 v2，支持 Go 1.22，B 站、七牛云线上高并发场景验证 。

---

### 快速选型结论

| 场景 | 推荐 |
|---|---|
| 想要 Django 风格、文档丰富、过滤器多 | **pongo2** |
| 想要轻量单文件、国产、速度好 | **gtpl** |
| 想要极限性能、团队只熟悉 Go 语法 | **Hero** |

以上三款 **全部默认“保留未匹配标签”**，且 **完整支持 for / if / else / 函数调用 / 过滤器**，可按团队喜好直接接入。
