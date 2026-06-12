import { LocalStorage } from 'quasar';
import { router } from 'src/router';
import { userState } from 'src/store/userState';

const clearAuthState = () => {
  userState.username = '';
  userState.accessToken = undefined;
  LocalStorage.remove('token');
};

const handleError = (error) => {
  let errorMessageText = error.data?.message || error.message || '网络错误';

  if (error.status === 401) {
    clearAuthState();
    errorMessageText = error.data?.message || '权限不足，请登录重试';
    if (router.currentRoute.value.path !== '/access/login') {
      router.push('/access/login');
    }
  } else {
    // eslint-disable-next-line no-console
    console.error('interceptor catch the error!\n', error);
  }

  return Promise.reject({
    error,
    message: errorMessageText,
  });
};

export default {
  onRequestRejected: (error) => handleError(error),
  onResponseFullFilled: (response) => {
    const { data } = response;
    if (data?.code && data?.code > 300) {
      return handleError(response);
    }
    return response;
  },
  onResponseRejected: (error) => handleError(error?.response || error),
};
