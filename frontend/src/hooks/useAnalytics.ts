import { useQuery } from '@tanstack/react-query';
import { api } from '../services/api';

export function useAnalytics(linkId?: string) {
  return useQuery({
    queryKey: ['analytics', linkId],
    queryFn: () => api.getAnalytics(linkId),
  });
}
