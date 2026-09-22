import { readFile, writeFile, unlink } from 'node:fs/promises'
import postcss from 'postcss'

const input = new URL('../dist/agent-chat.unscoped.css', import.meta.url)
const output = new URL('../dist/agent-chat.css', import.meta.url)
const root = postcss.parse(await readFile(input, 'utf8'))
const renamed = new Map()
root.walkAtRules('layer', (layer) => {
  if (layer.params !== 'theme') return
  layer.walkDecls((declaration) => {
    if (declaration.prop.startsWith('--')) renamed.set(declaration.prop, `--xgc-t3-${declaration.prop.slice(2)}`)
  })
})
root.walkAtRules('property', (property) => {
  renamed.set(property.params, `--xgc-t3-${property.params.slice(2)}`)
})
const rename = (value) => value.replace(/--[\w-]+/g, (name) => renamed.get(name) ?? name)
root.walkDecls((declaration) => {
  // Theme aliases consume host tokens, so they are appended after this pass.
  declaration.prop = rename(declaration.prop)
  declaration.value = rename(declaration.value)
})
const globals = []
root.walkAtRules('property', (property) => {
  property.params = rename(property.params)
  property.remove()
  globals.push(property)
})
const animations = new Map()
root.walkAtRules('keyframes', (keyframes) => {
  animations.set(keyframes.params, `xgc-t3-${keyframes.params}`)
  keyframes.params = animations.get(keyframes.params)
  keyframes.remove()
  globals.push(keyframes)
})
root.walkDecls((declaration) => {
  if (declaration.prop === 'animation' || declaration.prop === 'animation-name' || declaration.prop.startsWith('--')) {
    declaration.value = declaration.value.replace(/[a-zA-Z][\w-]*/g, (name) => animations.get(name) ?? name)
  }
})
root.walkRules((rule) => {
  rule.selector = rule.selector.replace(/(^|,\s*)(:root|:host|html|body)(?=\s|,|$)/g, '$1:scope')
})
const scope = postcss.atRule({ name: 'scope', params: '(.xgc-agent-chat)' })
scope.append(root.nodes)
scope.append(postcss.parse(await readFile(new URL('../src/t3-host-tokens.css', import.meta.url), 'utf8')).nodes)
const result = postcss.root()
result.append(postcss.comment({ text: 'T3 Code chat styles, scoped for XGC. See LICENSE.T3Code and UPSTREAM.json.' }))
result.append(globals)
result.append(scope)
result.append(postcss.parse(await readFile(new URL('../src/DecisionCard.css', import.meta.url), 'utf8')).nodes)
await writeFile(output, result.toString())
await unlink(input)
