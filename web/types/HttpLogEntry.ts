type HttpHeaders = Record<string, string[]>;

export type HttpLogEntry = {
  id: string;
  hostId: string;
  request: {
    host: string;
    url: string;
    method: string;
    headers: HttpHeaders;
    body: string;
    raw: string;
  };
  response: {
    statusCode: number;
    status: string;
    headers: HttpHeaders;
    body: string;
    raw: string;
  };
  createdAt: string;
};
