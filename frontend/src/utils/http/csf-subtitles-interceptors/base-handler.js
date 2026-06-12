const handleError = (error) => {
  let errorMessageText = error.data?.message || error.message || '网络错误';

  if (error.status === 401) {
    errorMessageText = error.data?.message || 'Token不可用';
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
    if ((data?.message && data?.message !== 'ok') || (data?.code && data?.code > 300)) {
      return handleError(response);
    }
    return response;
  },
  onResponseRejected: (error) => handleError(error?.response || error),
};
