package blog

import (
	"time"

	"github.com/unusualcodeorg/gomicro/blog-service/api/auth"
	"github.com/unusualcodeorg/gomicro/blog-service/api/blog/dto"
	"github.com/unusualcodeorg/gomicro/blog-service/api/blog/model"
	coredto "github.com/unusualcodeorg/goserve/arch/dto"
	"github.com/unusualcodeorg/goserve/arch/mongo"
	"github.com/unusualcodeorg/goserve/arch/network"
	"github.com/unusualcodeorg/goserve/arch/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service interface {
	SetBlogDtoCacheById(blog *dto.PublicBlog) error
	GetBlogDtoCacheById(id primitive.ObjectID) (*dto.PublicBlog, error)
	SetBlogDtoCacheBySlug(blog *dto.PublicBlog) error
	GetBlogDtoCacheBySlug(slug string) (*dto.PublicBlog, error)
	BlogSlugExists(slug string) bool
	GetPublisedBlogById(id primitive.ObjectID) (*dto.PublicBlog, error)
	GetPublishedBlogBySlug(slug string) (*dto.PublicBlog, error)
	getPublicPublishedBlog(filter bson.M) (*dto.PublicBlog, error)
	getPaginated(filter bson.M, p *coredto.Pagination, opts *options.FindOptions) ([]*dto.InfoBlog, error)
}

type service struct {
	network.BaseService
	blogQueryBuilder mongo.QueryBuilder[model.Blog]
	publicBlogCache  redis.Cache[dto.PublicBlog]
	authService      auth.Service
}

func NewService(db mongo.Database, store redis.Store, authService auth.Service) Service {
	return &service{
		BaseService:      network.NewBaseService(),
		blogQueryBuilder: mongo.NewQueryBuilder[model.Blog](db, model.CollectionName),
		publicBlogCache:  redis.NewCache[dto.PublicBlog](store),
		authService:      authService,
	}
}

func (s *service) SetBlogDtoCacheById(blog *dto.PublicBlog) error {
	key := "blog_" + blog.ID.Hex()
	return s.publicBlogCache.SetJSON(key, blog, time.Duration(10*time.Minute))
}

func (s *service) GetBlogDtoCacheById(id primitive.ObjectID) (*dto.PublicBlog, error) {
	key := "blog_" + id.Hex()
	return s.publicBlogCache.GetJSON(key)
}

func (s *service) SetBlogDtoCacheBySlug(blog *dto.PublicBlog) error {
	key := "blog_" + blog.Slug
	return s.publicBlogCache.SetJSON(key, blog, time.Duration(10*time.Minute))
}

func (s *service) GetBlogDtoCacheBySlug(slug string) (*dto.PublicBlog, error) {
	key := "blog_" + slug
	return s.publicBlogCache.GetJSON(key)
}

func (s *service) BlogSlugExists(slug string) bool {
	filter := bson.M{"slug": slug}
	projection := bson.D{{Key: "status", Value: 1}}
	opts := options.FindOne().SetProjection(projection)
	_, err := s.blogQueryBuilder.SingleQuery().FindOne(filter, opts)
	return err == nil
}

func (s *service) GetPublisedBlogById(id primitive.ObjectID) (*dto.PublicBlog, error) {
	filter := bson.M{"_id": id, "published": true, "status": true}
	return s.getPublicPublishedBlog(filter)
}

func (s *service) GetPublishedBlogBySlug(slug string) (*dto.PublicBlog, error) {
	filter := bson.M{"slug": slug, "published": true, "status": true}
	return s.getPublicPublishedBlog(filter)
}

func (s *service) getPublicPublishedBlog(filter bson.M) (*dto.PublicBlog, error) {
	projection := bson.D{{Key: "draftText", Value: 0}}
	opts := options.FindOne().SetProjection(projection)
	blog, err := s.blogQueryBuilder.SingleQuery().FindOne(filter, opts)
	if err != nil {
		return nil, network.NewNotFoundError("blog not found", err)
	}

	author, err := s.authService.FindUserPublicProfile(blog.Author)
	if err != nil {
		return nil, network.NewNotFoundError("author not found", err)
	}

	return dto.NewPublicBlog(blog, author)
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
