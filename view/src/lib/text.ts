/** A string for display: strings and numbers as they are, anything else (null, objects) as empty. */
export const asText = (value: unknown): string => {
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'bigint') return String(value);
  return '';
};
