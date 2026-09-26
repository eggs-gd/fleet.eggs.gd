export type Size = 's' | 'm' | 'l' | 'xl';

const SIZES: readonly Size[] = ['s', 'm', 'l', 'xl'];

export function isSize(value: string): value is Size {
  return (SIZES as readonly string[]).includes(value);
}

export function spaceVar(gap: Size | string): string {
  return isSize(gap) ? `var(--space-${gap})` : gap;
}
