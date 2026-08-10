// Hand-written type shims for @9am/fire-flame and @9am/fire-flame-react.
//
// Reason: both packages ship `dist/index.d.ts` and declare
// `"types": "./dist/index.d.ts"` in their package.json, but their `exports`
// field omits the `types` condition. TypeScript's `moduleResolution: "bundler"`
// honors `exports` strictly and refuses to resolve the .d.ts, falling back to
// "no declaration file". These shims re-declare the public surface area we use
// (FireFlameOption, FireFlame component, setOption ref) so tsc passes. The
// runtime resolution (Next.js bundler) is unaffected; the actual package is
// still loaded from node_modules.

// Hand-written type shims for @9am/fire-flame and @9am/fire-flame-react.
//
// Reason: both packages ship `dist/index.d.ts` and declare
// `"types": "./dist/index.d.ts"` in their package.json, but their `exports`
// field omits the `types` condition. TypeScript's `moduleResolution: "bundler"`
// honors `exports` strictly and refuses to resolve the .d.ts, falling back to
// "no declaration file". These shims re-declare the public surface area we use
// (FireFlameOption, FireFlame component, setOption ref) so tsc passes. The
// runtime resolution (Next.js bundler) is unaffected; the actual package is
// still loaded from node_modules.
//
// IMPORTANT: this file is a *script* (no top-level import/export), so the
// `declare module` statements are registered as ambient. If you add a
// top-level import, the ambient declarations stop working.

declare module '@9am/fire-flame' {
  export interface Point {
    x: number
    y: number
  }
  export interface MagDir {
    m: number
    d: number
  }
  export type VectorProps = Point | MagDir
  export class Vector {
    x: number
    y: number
    m: number
    d: number
    static add(a: Vector, b: Vector): Vector
    static subtract(a: Vector, b: Vector): Vector
    constructor(params: VectorProps)
    set(val: VectorProps): Vector
    add(v: Vector): Vector
    subtract(v: Vector): Vector
    multiply(scalar: number): Vector
    dot(v: Vector): number
  }
  export type PainterType = 'canvas' | 'svg'
  export type SizeCurveFunction = (x: number, prev: number) => number
  export interface FireFlameOption {
    x?: number
    y?: number
    mousemove?: boolean
    w?: number
    h?: number
    fps?: number
    wind?: Vector
    friction?: number
    particleNum?: number
    particleDistance?: number
    particleFPS?: number
    sizeCurveFn?: SizeCurveFunction
    innerColor?: string
    outerColor?: string
    painterType?: PainterType
    painter?: PainterType
  }
  export class FireFlame extends Vector {
    readonly container: HTMLElement
    static getDefaultOption(): FireFlameOption
    constructor(container: HTMLElement, option?: FireFlameOption)
    start(): void
    stop(): void
    setOption(option: FireFlameOption): void
    destroy(): void
  }
  export default FireFlame
}

declare module '@9am/fire-flame-react' {
  import type { FireFlameOption } from '@9am/fire-flame'
  export type { FireFlameOption, Vector } from '@9am/fire-flame'
  export type FireFlameProps = {
    children?: React.ReactNode
    option?: FireFlameOption
  }
  export const FireFlame: React.ForwardRefExoticComponent<
    FireFlameProps & React.RefAttributes<unknown>
  >
}
