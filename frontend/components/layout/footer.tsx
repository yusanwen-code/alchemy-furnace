import { SealDot } from '@/components/alchemy/float-card'

/** 宣纸风页脚：细边框、衬线落款、印章装饰 */
export function Footer() {
  return (
    <footer className="border-t border-border/70">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 py-10 sm:flex-row sm:justify-between sm:px-6">
        <p className="font-serif text-sm font-bold text-foreground">
          炼丹炉 <span className="mx-1 text-border">·</span>
          <span className="font-normal text-muted-foreground">炉中日月长，鼎内乾坤大</span>
        </p>
        <div className="flex items-center gap-2 text-xs text-sage">
          <SealDot />
          <span className="font-mono tracking-wide">Skill-Persona Alchemy</span>
        </div>
      </div>
    </footer>
  )
}
