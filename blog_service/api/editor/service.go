package editor

import (
	"time"

	"github.com/unusualcodeorg/gomicro/blog-service/api/auth"
	"github.com/unusualcodeorg/gomicro/blog-service/api/auth/message"
	authorDto "github.com/unusualcodeorg/gomicro/blog-service/api/author/dto"
	"github.com/unusualcodeorg/gomicro/blog-service/api/blog/dto"
	"github.com/unusualcodeorg/gomicro/blog-service/api/blog/model"
	coredto "github.com/unusualcodeorg/goserve/arch/dto"
	"github.com/unusualcodeorg/goserve/arch/mongo"
	"github.com/unusualcodeorg/goserve/arch/network"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service interface {
	GetBlogById(id primitive.ObjectID) (*authorDto.PrivateBlog, error)
	BlogPublication(blogId primitive.ObjectID, editor *message.User, publish bool) error
	GetPaginatedPublished(p *coredto.Pagination) ([]*dto.InfoBlog, error)
	GetPaginatedSubmitted(p *coredto.Pagination) ([]*dto.InfoBlog, error)
}

type service struct {
	network.BaseService
	blogQueryBuilder mongo.QueryBuilder[model.Blog]
	authService      auth.Service
}

func NewService(db mongo.Database, authService auth.Service) Service {
	return &service{
		BaseService:      network.NewBaseService(),
		blogQueryBuilder: mongo.NewQueryBuilder[model.Blog](db, model.CollectionName),
		authService:      authService,
	}
}

func (s *service) BlogPublication(blogId primitive.ObjectID, editor *message.User, publish bool) error {
	filter := bson.M{"_id": blogId, "status": true}
	blog, err := s.blogQueryBuilder.SingleQuery().FindOne(filter, nil)
	if err != nil {
		return network.NewNotFoundError("blog for _id "+blogId.Hex()+" not found", err)
	}

	if publish {
		if blog.Published {
			return network.NewBadRequestError("blog for _id "+blogId.Hex()+" is already published", err)
		}
		if !blog.Submitted {
			return network.NewBadRequestError("blog for _id "+blogId.Hex()+" is not submitted", err)
		}
	} else {
		if !blog.Published {
			return network.NewBadRequestError("blog for _id "+blogId.Hex()+" is not published", err)
		}
	}

	var update bson.M

	if publish {
		if blog.PublishedAt == nil {
			now := time.Now()
			blog.PublishedAt = &now
		}
		update = bson.M{"drafted": false, "submitted": false, "published": true, "text": blog.DraftText, "publishedAt": blog.PublishedAt}
	} else {
		update = bson.M{"drafted": true, "submitted": false, "published": false}
	}

	update["updatedBy"] = editor.ID
	update["updatedAt"] = time.Now()

	updated := bson.M{"$set": update}
	result, err := s.blogQueryBuilder.SingleQuery().UpdateOne(filter, updated)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return network.NewNotFoundError("blog not found", nil)
	}

	return nil
}

func (s *service) GetBlogById(id primitive.ObjectID) (*authorDto.PrivateBlog, error) {
	filter := bson.M{"_id": id, "status": true}
	blog, err := s.blogQueryBuilder.SingleQuery().FindOne(filter, nil)
	if err != nil {
		return nil, err
	}

	author, err := s.authService.FindUserPublicProfile(blog.Author)
	if err != nil {
		return nil, err
	}

	return authorDto.NewPrivateBlog(blog, author)
}

func (s *service) GetPaginatedPublished(p *coredto.Pagination) ([]*dto.InfoBlog, error) {
	filter := bson.M{"status": true, "published": true}
	return s.getPaginated(filter, p, nil)
}

func (s *service) GetPaginatedSubmitted(p *coredto.Pagination) ([]*dto.InfoBlog, error) {
	filter := bson.M{"status": true, "submitted": true}
	return s.getPaginated(filter, p, nil)
}

func (s *service) getPaginated(filter bson.M, p *coredto.Pagination, opts *options.FindOptions) ([]*dto.InfoBlog, error) {
	blogs, err := s.blogQueryBuilder.SingleQuery().FindPaginated(filter, p.Page, p.Limit, opts)
	if err != nil {
		return nil, err
	}

	dtos := make([]*dto.InfoBlog, len(blogs))

	for i, b := range blogs {
		d, err := dto.NewInfoBlog(b)
		if err != nil {
			return nil, err
		}
		dtos[i] = d
	}

	return dtos, nil
}
