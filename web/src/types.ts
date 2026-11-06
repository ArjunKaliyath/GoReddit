export interface User {
  id: number;
  username: string;
  karma: number;
}
export interface Community {
  id: number;
  name: string;
  description: string;
}
export interface Post {
  id: number;
  title: string;
  content: string;
  author_id: number;
  author_name: string;
  subreddit_id: number;
  subreddit_name: string;
  created_at: string;
  vote_count: { upvotes: number; downvotes: number };
}
export interface Comment {
  id: number;
  content: string;
  author_username: string;
  parent_comment_id: number | null;
}
export interface Message {
  id: number;
  from_username: string;
  content: string;
  created_at: string;
}
export type Action = (task: () => Promise<void>) => Promise<boolean>;
