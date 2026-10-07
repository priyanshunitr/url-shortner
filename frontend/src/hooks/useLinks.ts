import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import { Link } from '../types';

export function useLinks() {
  return useQuery({
    queryKey: ['links'],
    queryFn: api.getLinks,
  });
}

export function useCreateLink() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: Partial<Link>) => api.createLink(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['links'] });
    },
  });
}

export function useDeleteLink() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => api.deleteLink(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['links'] });
    },
  });
}
