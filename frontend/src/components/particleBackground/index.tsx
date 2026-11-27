import { useMemo } from 'react'

interface Particle {
    id: number
    size: number
    top: number
    left: number
    color: string
    duration: number
    delay: number
    blur: number
}

const COLORS = [
    'color-mix(in srgb, var(--ds-primary) 35%, transparent)',  // pink-400
    'color-mix(in srgb, var(--ds-secondary) 30%, transparent)',   // sky-400
    'color-mix(in srgb, var(--brand-lavender) 28%, transparent)',  // violet-400
    'color-mix(in srgb, var(--ds-primary) 20%, transparent)',
    'color-mix(in srgb, var(--ds-secondary) 20%, transparent)',
]

export default function ParticleBackground() {
    const particles = useMemo<Particle[]>(() => {
        return Array.from({ length: 22 }, (_, i) => ({
            id: i,
            size: 4 + Math.random() * 60,
            top: Math.random() * 100,
            left: Math.random() * 100,
            color: COLORS[Math.floor(Math.random() * COLORS.length)],
            duration: 6 + Math.random() * 10,
            delay: Math.random() * 8,
            blur: 2 + Math.random() * 20,
        }))
    }, [])

    return (
        <>
            <style>{`
                @keyframes floatUp {
                    0%   { transform: translateY(0px) scale(1);   opacity: 0; }
                    20%  { opacity: 1; }
                    80%  { opacity: 0.6; }
                    100% { transform: translateY(-60px) scale(1.1); opacity: 0; }
                }
            `}</style>
            <div className="fixed inset-0 z-1 pointer-events-none overflow-hidden">
                {particles.map(p => (
                    <div
                        key={p.id}
                        style={{
                            position: 'absolute',
                            width: p.size,
                            height: p.size,
                            top: `${p.top}%`,
                            left: `${p.left}%`,
                            borderRadius: '50%',
                            background: p.color,
                            filter: `blur(${p.blur}px)`,
                            animation: `floatUp ${p.duration}s ease-in-out ${p.delay}s infinite`,
                        }}
                    />
                ))}
            </div>
        </>
    )
}
