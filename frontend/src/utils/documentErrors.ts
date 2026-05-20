type DocumentErrorDetail = {
  order_id?: string;
  order_reference?: string;
  message?: string;
  reason?: string;
  code?: string;
};

const formatDocumentErrorObject = (data: any, fallback: string): string => {
  const base = typeof data?.error === 'string' && data.error.trim() ? data.error.trim() : fallback;
  const errors: DocumentErrorDetail[] = Array.isArray(data?.errors) ? data.errors : [];

  if (errors.length === 0) {
    return base;
  }

  const details = errors
    .map(error => {
      const identifier = error.order_reference || error.order_id || 'selected order';
      const reason = error.message || error.reason || error.code || 'cannot generate this document';
      return `${identifier}: ${reason}`;
    })
    .join('; ');

  return `${base}: ${details}`;
};

const parseDocumentErrorText = (text: string, fallback: string): string => {
  const trimmed = text.trim();
  if (!trimmed) return fallback;

  try {
    return formatDocumentErrorObject(JSON.parse(trimmed), fallback);
  } catch {
    return trimmed;
  }
};

export const documentErrorMessage = async (error: unknown, fallback: string): Promise<string> => {
  const responseData = (error as any)?.response?.data;

  if (typeof Blob !== 'undefined' && responseData instanceof Blob) {
    return parseDocumentErrorText(await responseData.text(), fallback);
  }

  if (typeof responseData === 'string') {
    return parseDocumentErrorText(responseData, fallback);
  }

  if (responseData && typeof responseData === 'object') {
    return formatDocumentErrorObject(responseData, fallback);
  }

  const message = (error as any)?.message;
  return typeof message === 'string' && message.trim() ? `${fallback}: ${message}` : fallback;
};
