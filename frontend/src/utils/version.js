const HIDDEN_VERSION_PATTERN = /^(unknown|unknow|dev|development)(\s+lite)?$/i;

export const normalizeDisplayVersion = (version) => {
  if (!version) return '';

  const normalized = String(version).trim();
  if (!normalized) return '';
  if (HIDDEN_VERSION_PATTERN.test(normalized)) return '';

  return normalized;
};
