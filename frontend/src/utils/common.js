import { PROJECT_BUG_TEMPLATE_URL } from 'src/constants/ProjectLinks';

export const deepCopy = (obj) => JSON.parse(JSON.stringify(obj));

export const gotoGithubIssuePage = () => {
  window.open(PROJECT_BUG_TEMPLATE_URL, '_blank');
};

export const isImdbId = (str) => {
  if (!str) return false;
  if (str === 'tt00000') return false;
  return /^tt\d{7,8}$/.test(str);
};
